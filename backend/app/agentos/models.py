"""AgentOS 持久化模型"""
from datetime import datetime

from sqlalchemy import Column, DateTime, Integer, String, Text
from sqlalchemy import func as sa_func

from app.database import Base


class AgentOSOperationLog(Base):
    """AgentOS 操作审计日志"""
    __tablename__ = "agentos_operation_log"

    id = Column(Integer, primary_key=True, autoincrement=True)
    user_id = Column(Integer, nullable=False, index=True)
    item_id = Column(String(128), nullable=False, comment="WorkItem ID (e.g. exception:42)")
    action = Column(String(32), nullable=False, comment="approve / reject / status_update")
    source_type = Column(String(32), nullable=True, comment="agent_action / exception / notification / listing_task")
    previous_status = Column(String(32), nullable=True)
    new_status = Column(String(32), nullable=True)
    comment = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
