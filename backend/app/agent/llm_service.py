"""Agent LLM 解释服务

为 Agent 决策生成自然语言解释文案。
支持多提供商（OpenAI / Anthropic / Ollama），可从 DB 配置读取。
无 Key 时回退到模板文案。
"""
import json
import logging
from typing import Any, Optional
import httpx
from app.config import settings

logger = logging.getLogger(__name__)


AGENT_PROMPTS = {
    "A5": {
        "system": "你是一个跨境电商库存管理专家。请根据库存预警数据，用自然语言解释库存状态、风险和推荐操作。简洁、专业、中文。",
        "template": (
            "库存状态：{status}\nSKU：{sku}\n可售库存：{sellable} 件\n"
            "在途库存：{transit} 件\n日均销量：{daily_sales} 件/天\n"
            "可售天数：{sellable_days} 天\n采购提前期：{lead_time} 天\n"
            "安全库存天数：{safety_days} 天\n风险原因：{risk_reason}\n"
            "请用一段话解释当前库存状态、风险严重程度和紧急程度。"
        ),
    },
    "G3": {
        "system": "你是一个跨境电商折扣风控专家。请根据折扣检查结果，用自然语言解释风险和决策原因。简洁、专业、中文。",
        "template": (
            "SKU：{sku}\n原价：¥{original_price}\n成本价：¥{cost_price}\n"
            "折扣详情：{discount_details}\n折后价：¥{final_price}\n"
            "毛利：¥{gross_profit}\n毛利率：{gross_margin}%\n"
            "决策：{action}\n原因：{reason}\n"
            "请用一段话解释这个折扣决策的原因和风险。"
        ),
    },
    "A6": {
        "system": "你是一个跨境电商利润分析专家。请根据利润检查结果，用自然语言解释利润状况和风险。简洁、专业、中文。",
        "template": (
            "SKU：{sku}\n售价：¥{selling_price}\n成本：¥{cost_price}\n"
            "费用明细：{fee_breakdown}\n净利润：¥{profit}\n"
            "毛利率：{margin}%\n是否亏损：{is_loss}\n"
            "异常原因：{anomaly_reason}\n"
            "请用一段话解释该SKU的利润状况和需要关注的风险。"
        ),
    },
    "A3": {
        "system": "你是一个跨境电商广告投放专家。请根据广告分析结果，用自然语言解释ACoS状况和优化建议。简洁、专业、中文。",
        "template": (
            "广告活动：{campaign_id}\n花费：¥{spend}\n销售额：¥{sales}\n"
            "ACoS：{acos}%\n目标ACoS：{target_acos}%\n"
            "点击率：{ctr}%\n转化率：{cvr}%\n状态：{status}\n"
            "请用一段话解释广告效果和优化建议。"
        ),
    },
}


class AgentLlmService:

    @staticmethod
    async def explain(agent_id: str, summary: dict, db: Any = None) -> str:
        """生成自然语言解释

        优先从 DB 读取 LLM 配置，支持各 Agent 独立模型。
        无 Key 时回退到模板文案。
        """
        prompt_config = AGENT_PROMPTS.get(agent_id)
        if not prompt_config:
            return ""

        # 读取 LLM 配置
        if db is not None:
            from app.agent.config_service import ConfigService
            llm_config = await ConfigService.get_llm_config(db)
            agent_overrides = llm_config.get("agent_overrides", {})
            model = agent_overrides.get(agent_id, llm_config["model"])
            provider = llm_config["provider"]
            api_key = llm_config["api_key"]
            base_url = llm_config["base_url"]
        else:
            # 从环境变量回退
            provider = "openai"
            model = settings.LLM_MODEL
            api_key = settings.LLM_API_KEY
            base_url = settings.LLM_API_URL

        # 无 Key → 模板回退
        if not api_key or api_key == "ollama":
            return AgentLlmService._fallback_explanation(agent_id, summary)

        # 构造 prompt
        prompt_text = prompt_config["template"].format(**summary)
        system_prompt = prompt_config["system"]

        try:
            if provider == "anthropic":
                return await AgentLlmService._call_anthropic(api_key, model, system_prompt, prompt_text)
            else:
                return await AgentLlmService._call_openai(api_key, base_url, model, system_prompt, prompt_text)
        except Exception as e:
            logger.warning("LLM 调用失败 (%s), 回退模板: %s", provider, e)
            return AgentLlmService._fallback_explanation(agent_id, summary)

    @staticmethod
    async def _call_openai(api_key: str, base_url: str, model: str, sys: str, user: str) -> str:
        payload = {
            "model": model,
            "messages": [{"role": "system", "content": sys}, {"role": "user", "content": user}],
            "temperature": 0.7, "max_tokens": 300,
        }
        headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(base_url + "/chat/completions", json=payload, headers=headers)
            resp.raise_for_status()
            return resp.json()["choices"][0]["message"]["content"].strip()

    @staticmethod
    async def _call_anthropic(api_key: str, model: str, sys: str, user: str) -> str:
        payload = {
            "model": model,
            "system": sys,
            "messages": [{"role": "user", "content": user}],
            "max_tokens": 300,
        }
        headers = {"x-api-key": api_key, "anthropic-version": "2023-06-01", "Content-Type": "application/json"}
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post("https://api.anthropic.com/v1/messages", json=payload, headers=headers)
            resp.raise_for_status()
            return resp.json()["content"][0]["text"].strip()

    @staticmethod
    def _fallback_explanation(agent_id: str, summary: dict) -> str:
        """模板回退"""
        if agent_id == "A5":
            s, sku, days, reason = summary.get("status", ""), summary.get("sku", ""), summary.get("sellable_days", "?"), summary.get("risk_reason", "")
            if s == "red": return f"⚠️ SKU {sku} 库存告急！仅可售 {days} 天。{reason}。建议立即安排紧急补货，优先选择空运。"
            if s == "yellow": return f"⚡ SKU {sku} 库存预警，可售 {days} 天。{reason}。建议尽快安排补货。"
            return f"✅ SKU {sku} 库存充足，可售 {days} 天。"
        if agent_id == "G3":
            act, sku, price, cost, reason = summary.get("action", ""), summary.get("sku", ""), summary.get("final_price", "?"), summary.get("cost_price", "?"), summary.get("reason", "")
            if act == "block": return f"🚫 SKU {sku} 折扣已阻断！折后价 ¥{price} 低于成本 ¥{cost}。{reason}"
            if act == "warn": return f"⚠️ SKU {sku} 折扣需要关注。{reason}"
            return f"✅ SKU {sku} 折扣检查通过，折后价 ¥{price}。"
        if agent_id == "A6":
            sku, profit, margin, loss = summary.get("sku", ""), summary.get("profit", "?"), summary.get("margin", "?"), summary.get("is_loss", False)
            if loss: return f"🔴 SKU {sku} 亏损！单件亏 ¥{abs(float(profit)):.2f}，毛利率 {margin}%。建议提价或降成本。"
            if margin != "?" and float(margin) < 15: return f"🟡 SKU {sku} 毛利率偏低 ({margin}%)。建议优化成本。"
            return f"✅ SKU {sku} 利润正常，毛利率 {margin}%。"
        if agent_id == "A3":
            cid, acos, st = summary.get("campaign_id", ""), summary.get("acos", "?"), summary.get("status", "")
            if st == "critical": return f"🔴 广告 {cid} 严重亏损！ACoS {acos}% 超过毛利率，建议暂停。"
            if st == "warning": return f"🟡 广告 {cid} ACoS ({acos}%) 偏高，建议优化。"
            return f"✅ 广告 {cid} 效果正常，ACoS {acos}%。"
        return ""
