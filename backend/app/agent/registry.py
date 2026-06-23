"""Agent 注册中心"""

from typing import Optional
from app.agent.base import BaseAgent


class AgentRegistry:
    _agents: dict[str, type[BaseAgent]] = {}

    @classmethod
    def register(cls, agent_cls: type[BaseAgent]) -> type[BaseAgent]:
        if not agent_cls.agent_id:
            raise ValueError(f"Agent {agent_cls.__name__} must define agent_id")
        cls._agents[agent_cls.agent_id] = agent_cls
        return agent_cls

    @classmethod
    def get_agent_class(cls, agent_id: str) -> Optional[type[BaseAgent]]:
        return cls._agents.get(agent_id)

    @classmethod
    def list_agents(cls) -> list[dict[str, str]]:
        return [
            {
                "agent_id": a.agent_id,
                "name": a.name,
                "description": a.description,
                "decision_points": a.decision_points,
                "version": a.version,
            }
            for a in cls._agents.values()
        ]

    @classmethod
    def get_metadata(cls, agent_id: str) -> Optional[dict]:
        a = cls._agents.get(agent_id)
        if not a:
            return None
        return {
            "agent_id": a.agent_id,
            "name": a.name,
            "description": a.description,
            "decision_points": a.decision_points,
            "version": a.version,
        }


def register_agent(cls):
    return AgentRegistry.register(cls)
