"""Agent LLM 解释与分析服务

为 Agent 决策生成自然语言解释文案，以及结构化决策分析。
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
    "A2": {
        "system": "你是一个跨境电商 Listing 优化专家。请根据优化结果，用自然语言解释标题和关键词策略。简洁、专业、中文。",
        "template": (
            "产品：{product_name}\n市场：{marketplace}\n"
            "原标题：{original_title}\n优化标题：{optimized_title}\n"
            "关键词数：{keyword_count}\n优化建议：{suggestions}\n"
            "请用一段话解释此次优化策略。"
        ),
    },
    "A4": {
        "system": "你是一个跨境电商客服专家。请根据客户消息内容，用自然语言解释客服处理建议。简洁、专业、中文。",
        "template": (
            "客户消息：{message}\n识别意图：{intent}\n"
            "建议回复：{reply_suggestion}\n置信度：{confidence}\n"
            "请用一段话解释客户意图和处理建议。"
        ),
    },
    "A1": {
        "system": "你是一个跨境电商选品专家。请根据选品分析结果，用自然语言解释选品建议和市场机会。简洁、专业、中文。",
        "template": (
            "品类：{category}\n市场：{marketplace}\n"
            "候选产品数：{candidate_count}\n最高分：{top_score}\n"
            "请用一段话解释选品策略和市场机会。"
        ),
    },
    "A7": {
        "system": "你是一个跨境电商合规专家。请根据合规检查结果，用自然语言解释法规风险和认证要求。简洁、专业、中文。",
        "template": (
            "产品：{product}\n品类：{category}\n目标国家：{country}\n"
            "风险等级：{risk_level}\n认证要求：{certifications}\n"
            "限制条件：{restrictions}\n"
            "请用一段话解释该产品的合规风险和注意事项。"
        ),
    },
    "G2": {
        "system": "你是一个跨境仓储物流专家。请根据仓库和通关建议，用自然语言解释物流策略和通关要求。简洁、专业、中文。",
        "template": (
            "产品：{product}\n目的地：{destination}\n"
            "HS 编码：{hs_code}\n所需文件：{documents}\n"
            "建议策略：{strategy}\n"
            "请用一段话解释仓储物流建议。"
        ),
    },
    "G1": {
        "system": "你是一个电商运营驾驶舱专家。请根据运营概览数据，用自然语言总结当前运营状态和关键风险。简洁、专业、中文。",
        "template": (
            "产品总数：{product_count}\nSKU 总数：{sku_count}\n"
            "待处理预警：{alert_count}\n数据时间：{timestamp}\n"
            "请用一段话总结当前运营状态。"
        ),
    },
}

# ── 结构化分析 Prompt 配置 ──────────────────────────────────────
# 与 AGENT_PROMPTS（用于解释）不同，这里要求 LLM 输出 JSON，
# 直接参与决策判断。
ANALYSIS_PROMPTS = {
    "A5": {
        "stock_alert": {
            "system": (
                "你是一个跨境电商库存管理专家。你的任务是根据库存数据做出专业判断，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下 SKU 的库存状况：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "risk_level": "low|medium|high",\n'
                '  "risk_reason": "风险分析（中文，考虑季节、趋势、供应商因素）",\n'
                '  "suggested_replenish_qty": 建议补货数量（整数，0表示不需要补货）, \n'
                '  "suggested_logistics": "sea_freight|express_sea|air_freight|air_express",\n'
                '  "additional_notes": "其他值得关注的问题或建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "G3": {
        "discount_check": {
            "system": (
                "你是一个跨境电商折扣风控专家。根据折扣叠加数据判断风险，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下折扣叠加风险：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "action": "allow|warn|block",\n'
                '  "risk_level": "low|medium|high",\n'
                '  "risk_reason": "风险原因（中文，考虑叠加幅度、平台政策等）",\n'
                '  "suggested_logistics_adjustment": "有无物流调整建议",\n'
                '  "additional_notes": "其他建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A6": {
        "profit_check": {
            "system": (
                "你是一个跨境电商利润分析专家。根据成本和售价数据判断利润风险，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下 SKU 的利润状况：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "is_loss": true|false,\n'
                '  "risk_level": "low|medium|high",\n'
                '  "risk_reason": "利润分析（中文，考虑费用构成）",\n'
                '  "cost_suggestion": "成本优化建议（中文）",\n'
                '  "additional_notes": "其他值得关注的异常（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A3": {
        "acos_analysis": {
            "system": (
                "你是一个跨境电商广告投放专家。根据广告数据判断 ACoS 风险，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下广告活动的 ACoS 状况：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "status": "normal|warning|critical",\n'
                '  "root_cause": "根本原因诊断（中文）",\n'
                '  "bid_suggestion": "出价调整建议（中文）",\n'
                '  "keyword_suggestion": "关键词优化建议（中文）",\n'
                '  "additional_notes": "其他建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A2": {
        "listing_optimize": {
            "system": (
                "你是一个跨境电商 Listing 优化专家。根据产品信息和关键词数据给出优化建议，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请优化以下产品 Listing：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "title_suggestion": "优化标题（中文说明策略）",\n'
                '  "bullet_points": "五点描述建议（中文）",\n'
                '  "keyword_strategy": "关键词策略（中文）",\n'
                '  "additional_notes": "其他优化建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A4": {
        "auto_reply": {
            "system": (
                "你是一个跨境电商客服专家。根据客户消息内容判断意图并给出回复建议，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下客户消息：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "intent": "客户意图分类（中文）",\n'
                '  "reply_suggestion": "建议回复内容（中文）",\n'
                '  "sentiment": "positive|neutral|negative",\n'
                '  "requires_human": true|false,\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A1": {
        "product_scout": {
            "system": (
                "你是一个跨境电商选品专家。根据产品候选数据给出选品建议，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下选品数据：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "top_product": "推荐产品名称",\n'
                '  "market_insight": "市场洞察（中文）",\n'
                '  "risk_flags": "风险提示（中文）",\n'
                '  "scoring_approach": "评分策略建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "A7": {
        "compliance_check": {
            "system": (
                "你是一个跨境电商合规专家。根据产品描述和目标市场判断合规风险，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下产品的合规风险：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "risk_level": "low|medium|high",\n'
                '  "risk_reason": "合规风险分析（中文）",\n'
                '  "required_certifications": "所需认证列表（逗号分隔）",\n'
                '  "concerns": "值得关注的合规问题（中文）",\n'
                '  "additional_notes": "其他建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "G2": {
        "customs_clearance": {
            "system": (
                "你是一个跨境物流专家。根据产品信息和目的地判断通关风险和物流策略，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下产品的通关和物流需求：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "hs_code_suggestion": "建议 HS 编码",\n'
                '  "risk_level": "low|medium|high",\n'
                '  "risk_reason": "通关风险分析（中文）",\n'
                '  "documents": "所需清关文件（逗号分隔）",\n'
                '  "additional_notes": "其他建议（中文）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
    },
    "G1": {
        "dashboard_overview": {
            "system": (
                "你是一个电商运营驾驶舱专家。根据运营概览数据总结运营状态，"
                "输出严格的 JSON 格式结果，不要包含任何其他文字。"
            ),
            "template": (
                "请分析以下运营概览数据：\n\n"
                "{context}\n\n"
                "请输出严格的 JSON（不要 markdown 代码块标记）：\n"
                "{{\n"
                '  "summary": "运营整体状况摘要（中文）",\n'
                '  "key_risks": "主要风险点（中文，无风险可填无）",\n'
                '  "confidence": 0.0-1.0\n'
                "}}"
            ),
        },
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
                return await AgentLlmService._call_anthropic(
                    api_key, model, system_prompt, prompt_text
                )
            else:
                return await AgentLlmService._call_openai(
                    api_key, base_url, model, system_prompt, prompt_text
                )
        except Exception as e:
            logger.warning("LLM 调用失败 (%s), 回退模板: %s", provider, e)
            return AgentLlmService._fallback_explanation(agent_id, summary)

    @staticmethod
    async def _call_openai(
        api_key: str, base_url: str, model: str, sys: str, user: str
    ) -> str:
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": sys},
                {"role": "user", "content": user},
            ],
            "temperature": 0.7,
            "max_tokens": 300,
        }
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(
                base_url + "/chat/completions", json=payload, headers=headers
            )
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
        headers = {
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(
                "https://api.anthropic.com/v1/messages", json=payload, headers=headers
            )
            resp.raise_for_status()
            return resp.json()["content"][0]["text"].strip()

    @staticmethod
    def _fallback_explanation(agent_id: str, summary: dict) -> str:
        """模板回退"""
        if agent_id == "A5":
            s, sku, days, reason = (
                summary.get("status", ""),
                summary.get("sku", ""),
                summary.get("sellable_days", "?"),
                summary.get("risk_reason", ""),
            )
            if s == "red":
                return f"⚠️ SKU {sku} 库存告急！仅可售 {days} 天。{reason}。建议立即安排紧急补货，优先选择空运。"
            if s == "yellow":
                return f"⚡ SKU {sku} 库存预警，可售 {days} 天。{reason}。建议尽快安排补货。"
            return f"✅ SKU {sku} 库存充足，可售 {days} 天。"
        if agent_id == "G3":
            act, sku, price, cost, reason = (
                summary.get("action", ""),
                summary.get("sku", ""),
                summary.get("final_price", "?"),
                summary.get("cost_price", "?"),
                summary.get("reason", ""),
            )
            if act == "block":
                return f"🚫 SKU {sku} 折扣已阻断！折后价 ¥{price} 低于成本 ¥{cost}。{reason}"
            if act == "warn":
                return f"⚠️ SKU {sku} 折扣需要关注。{reason}"
            return f"✅ SKU {sku} 折扣检查通过，折后价 ¥{price}。"
        if agent_id == "A6":
            sku, profit, margin, loss = (
                summary.get("sku", ""),
                summary.get("profit", "?"),
                summary.get("margin", "?"),
                summary.get("is_loss", False),
            )
            if loss:
                return f"🔴 SKU {sku} 亏损！单件亏 ¥{abs(float(profit)):.2f}，毛利率 {margin}%。建议提价或降成本。"
            if margin != "?" and float(margin) < 15:
                return f"🟡 SKU {sku} 毛利率偏低 ({margin}%)。建议优化成本。"
            return f"✅ SKU {sku} 利润正常，毛利率 {margin}%。"
        if agent_id == "A3":
            cid, acos, st = (
                summary.get("campaign_id", ""),
                summary.get("acos", "?"),
                summary.get("status", ""),
            )
            if st == "critical":
                return f"🔴 广告 {cid} 严重亏损！ACoS {acos}% 超过毛利率，建议暂停。"
            if st == "warning":
                return f"🟡 广告 {cid} ACoS ({acos}%) 偏高，建议优化。"
            return f"✅ 广告 {cid} 效果正常，ACoS {acos}%。"
        if agent_id == "A2":
            name = summary.get("product_name", "")
            orig = summary.get("original_title", "")
            opt = summary.get("optimized_title", "")
            if opt and opt != orig:
                return f"📝 {name} 标题已优化。原: {orig[:40]}… → 新: {opt[:40]}…"
            return f"📝 {name} Listing 当前标题可维持。"
        if agent_id == "A4":
            msg = summary.get("message", "")
            intent = summary.get("intent", "")
            if intent:
                return f"💬 客户意图: {intent}。消息: {msg[:50]}…"
            return '💬 收到客户消息，建议人工处理。'
        if agent_id == "A1":
            cat = summary.get("category", "")
            n = summary.get("candidate_count", 0)
            score = summary.get("top_score", "?")
            return f"🔍 品类 {cat} 选品分析完成，扫描 {n} 个产品，最高分 {score}。"
        if agent_id == "A7":
            p = summary.get("product", "")
            risk = summary.get("risk_level", "?")
            certs = summary.get("certifications", "")
            return f"🔒 {p} 合规检查完成，风险等级 {risk}，认证要求: {certs}"
        if agent_id == "G1":
            n = summary.get("alert_count", 0)
            product_c = summary.get("product_count", "?")
            return f"📊 运营概览: {product_c} 个产品, {n} 条待处理预警"
        if agent_id == "G2":
            dest = summary.get("destination", "")
            hs = summary.get("hs_code", "?")
            return f"📦 发往 {dest} 的物流建议已生成，HS 编码: {hs}"
        return ""

    # ── 结构化分析（用于 LLM 直接参与决策）────────────────────

    @staticmethod
    async def analyze(
        agent_id: str,
        decision_point: str,
        context: dict[str, Any],
        db: Any = None,
    ) -> Optional[dict[str, Any]]:
        """调用 LLM 进行结构化分析，返回 JSON dict

        与 explain() 不同，此方法返回结构化数据而非自然语言文本，
        用于 LLM 直接参与决策判断。

        成功时返回解析后的 JSON dict，失败/无 Key 时返回 None。
        永远不抛异常——调用方负责兜底。
        """
        analysis_cfg = ANALYSIS_PROMPTS.get(agent_id, {}).get(decision_point)
        if not analysis_cfg:
            return None

        # 读取 LLM 配置
        if db is not None:
            from app.agent.config_service import ConfigService

            try:
                llm_config = await ConfigService.get_llm_config(db)
                agent_overrides = llm_config.get("agent_overrides", {})
                model = agent_overrides.get(agent_id, llm_config["model"])
                provider = llm_config["provider"]
                api_key = llm_config["api_key"]
                base_url = llm_config["base_url"]
            except Exception:
                return None
        else:
            provider = "openai"
            model = settings.LLM_MODEL
            api_key = settings.LLM_API_KEY
            base_url = settings.LLM_API_URL

        # 无 Key → 无法分析
        if not api_key or api_key == "ollama":
            return None

        # 构造 context 字符串
        context_lines = "\n".join(f"{k}: {v}" for k, v in context.items())
        prompt_text = analysis_cfg["template"].format(context=context_lines)
        system_prompt = analysis_cfg["system"]

        try:
            if provider == "anthropic":
                raw = await AgentLlmService._call_anthropic_json(
                    api_key, model, system_prompt, prompt_text
                )
            else:
                raw = await AgentLlmService._call_openai_json(
                    api_key, base_url, model, system_prompt, prompt_text
                )
            if not raw:
                return None

            # 解析 JSON（兼容首尾有多余文字的情况）
            parsed = AgentLlmService._parse_json_response(raw)
            if parsed is None:
                logger.warning(
                    "LLM 分析返回非 JSON: agent=%s dp=%s", agent_id, decision_point
                )
                return None

            return parsed
        except Exception as e:
            logger.warning("LLM 分析失败 (%s): %s", provider, e)
            return None

    @staticmethod
    async def _call_openai_json(
        api_key: str, base_url: str, model: str, sys: str, user: str
    ) -> Optional[str]:
        """调用 OpenAI 兼容 API，返回原始文本"""
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": sys},
                {"role": "user", "content": user},
            ],
            "temperature": 0.3,
            "max_tokens": 600,
        }
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }
        try:
            async with httpx.AsyncClient(timeout=15) as client:
                resp = await client.post(
                    base_url + "/chat/completions", json=payload, headers=headers
                )
                resp.raise_for_status()
                return resp.json()["choices"][0]["message"]["content"].strip()
        except Exception:
            return None

    @staticmethod
    async def _call_anthropic_json(
        api_key: str, model: str, sys: str, user: str
    ) -> Optional[str]:
        """调用 Anthropic API，返回原始文本"""
        payload = {
            "model": model,
            "system": sys,
            "messages": [{"role": "user", "content": user}],
            "max_tokens": 600,
        }
        headers = {
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        }
        try:
            async with httpx.AsyncClient(timeout=15) as client:
                resp = await client.post(
                    "https://api.anthropic.com/v1/messages",
                    json=payload,
                    headers=headers,
                )
                resp.raise_for_status()
                return resp.json()["content"][0]["text"].strip()
        except Exception:
            return None

    @staticmethod
    def _parse_json_response(raw: str) -> Optional[dict[str, Any]]:
        """从 LLM 返回文本中提取 JSON"""
        # 移除 markdown 代码块
        text = raw.strip()
        if text.startswith("```"):
            # 移除 ```json 或 ``` 包裹
            first_line = text.index("\n")
            text = text[first_line:].strip()
        if text.endswith("```"):
            text = text[:-3].strip()

        try:
            return json.loads(text)
        except json.JSONDecodeError:
            # 尝试找第一个 { 和最后一个 }
            start = text.find("{")
            end = text.rfind("}")
            if start != -1 and end != -1 and end > start:
                try:
                    return json.loads(text[start : end + 1])
                except json.JSONDecodeError:
                    pass
            return None
