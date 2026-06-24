"""A7 合规检测 Agent (Phase 2 — LLM 增强版)

设计依据: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent7

Phase 2 改进：LLM 参与合规风险分析，规则表降级为安全网。
- 优先调用 LLM 从产品描述分析合规风险
- LLM 失败时自动降级为规则表兜底

Phase 1 原始设计：
- 产品合规检查：认证要求、禁限售、标签要求、税务合规
- 输入产品信息和目标市场，输出合规风险和所需认证清单
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.data_service import AgentDataService
from app.agent.llm_service import AgentLlmService

REQUIRED = ["product_name", "category", "target_country", "target_platform"]

COMPLIANCE_RULES = {
    ("electronics", "US"): {
        "certifications": ["FCC", "UL"],
        "restrictions": [],
        "risk": "medium",
    },
    ("electronics", "EU"): {
        "certifications": ["CE", "RoHS", "WEEE"],
        "restrictions": [],
        "risk": "medium",
    },
    ("electronics", "UK"): {
        "certifications": ["UKCA", "RoHS"],
        "restrictions": [],
        "risk": "medium",
    },
    ("electronics", "JP"): {
        "certifications": ["PSE", "MIC"],
        "restrictions": [],
        "risk": "medium",
    },
    ("baby", "US"): {
        "certifications": ["CPC", "ASTM F963"],
        "restrictions": [],
        "risk": "high",
    },
    ("baby", "EU"): {
        "certifications": ["CE", "EN 71"],
        "restrictions": [],
        "risk": "high",
    },
    ("baby", "UK"): {
        "certifications": ["UKCA", "EN 71"],
        "restrictions": [],
        "risk": "high",
    },
    ("cosmetics", "EU"): {
        "certifications": ["CPNP"],
        "restrictions": ["动物测试"],
        "risk": "high",
    },
    ("cosmetics", "US"): {
        "certifications": ["FDA"],
        "restrictions": [],
        "risk": "medium",
    },
    ("food", "US"): {"certifications": ["FDA"], "restrictions": [], "risk": "high"},
    ("food", "EU"): {
        "certifications": ["EU Novel Food"],
        "restrictions": [],
        "risk": "high",
    },
    ("toys", "US"): {
        "certifications": ["CPC", "ASTM F963"],
        "restrictions": [],
        "risk": "high",
    },
    ("toys", "EU"): {
        "certifications": ["CE", "EN 71"],
        "restrictions": [],
        "risk": "high",
    },
}


def _sf(v: Any, d: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return d


def _missing(c: dict, r: list) -> list:
    return [f for f in r if f not in c or c[f] is None]


@register_agent
class A7ComplianceGuardAgent(BaseAgent):
    agent_id = "A7"
    name = "合规检测 Agent"
    description = "跨境电商产品合规检查：认证要求、禁限售、标签合规、税务检查"
    decision_points = ["compliance_check", "certification_lookup"]
    version = "1.0.0"
    DEFAULT_STAGES = {
        "compliance_check": EvolutionStage.SEMI_AUTONOMOUS,
        "certification_lookup": EvolutionStage.SUGGESTION,
    }

    async def decide(self, point: str, ctx: dict, db: Any = None) -> dict:
        if db is not None and point == "compliance_check":
            ctx = await AgentDataService.fill_product_context(db, ctx)
        if point == "compliance_check":
            return await self._check_with_llm(ctx, db=db)
        if point == "certification_lookup":
            return self._lookup(ctx)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 合规检查（LLM 增强版）
    # ──────────────────────────────
    async def _check_with_llm(self, ctx: dict, db: Any = None) -> dict:
        """合规检查主入口"""
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("compliance_check", miss)

        # ① 公式兜底
        formula_result = self._formula_check(ctx)

        # ② LLM 分析
        llm_decision = None
        stage = self.get_stage("compliance_check")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "product_name": ctx.get("product_name", ""),
                "category": ctx.get("category", ""),
                "target_country": ctx.get("target_country", ""),
                "target_platform": ctx.get("target_platform", ""),
                "description": ctx.get("description", ctx.get("product_name", "")),
                "formula_risk": formula_result.get("risk_level", "unknown"),
                "formula_certs": ", ".join(
                    formula_result.get("required_certifications", [])
                ),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A7", "compliance_check", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ③ 仲裁
        if llm_decision:
            result = {
                **formula_result,
                "risk_reason": llm_decision.get("risk_reason", ""),
                "concerns": llm_decision.get("concerns", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "risk_reason": "",
                "concerns": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ④ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A7",
                {
                    "product": result.get("product", ""),
                    "category": result.get("category", ""),
                    "country": result.get("country", ""),
                    "risk_level": result.get("risk_level", ""),
                    "certifications": ", ".join(
                        result.get("required_certifications", [])
                    ),
                    "restrictions": ", ".join(result.get("restrictions", [])),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式兜底
    # ──────────────────────────────
    def _formula_check(self, ctx: dict) -> dict:
        """纯规则表合规检查，作为 LLM 失败时的兜底"""
        name = str(ctx.get("product_name", ""))
        cat = str(ctx.get("category", "")).lower().strip()
        country = str(ctx.get("target_country", "")).upper().strip()
        platform = str(ctx.get("target_platform", "")).lower().strip()

        rule = COMPLIANCE_RULES.get((cat, country))
        if not rule:
            rule = {
                "certifications": [],
                "restrictions": ["请人工核实具体认证要求"],
                "risk": "unknown",
            }

        certs = rule["certifications"]
        restrictions = rule["restrictions"]
        risk = rule["risk"]

        blocked_platforms = []
        if (
            platform == "amazon"
            and cat == "electronics"
            and country == "US"
            and "FCC" not in certs
        ):
            blocked_platforms.append("Amazon US (FCC 要求)")

        return {
            "product": name,
            "category": cat,
            "country": country,
            "platform": platform,
            "required_certifications": certs,
            "restrictions": restrictions,
            "risk_level": risk,
            "blocked_platforms": blocked_platforms,
            "confidence": 0.90 if risk != "unknown" else 0.60,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        risk = result.get("risk_level")
        if risk is not None and risk not in ("low", "medium", "high"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    def _lookup(self, ctx: dict) -> dict:
        cert = ctx.get("certification", "")
        country = ctx.get("country", "US")
        known = {
            "FCC": "FCC 认证（美国）：电子产品的电磁兼容性强制认证",
            "CE": "CE 认证（欧盟）：产品安全、健康、环保的基本要求",
            "CPC": "CPC 认证（美国）：儿童产品安全强制认证",
            "FDA": "FDA 认证（美国）：食品、药品、化妆品的 FDA 注册",
        }
        return {
            "certification": cert,
            "country": country,
            "description": known.get(
                cert.upper(), f"请人工核实 {cert} 在 {country} 的具体要求"
            ),
            "confidence": 0.85 if cert.upper() in known else 0.50,
        }

    def _insufficient(self, p: str, m: list) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
