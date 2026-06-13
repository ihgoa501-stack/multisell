"""API密钥加密/解密工具"""

from cryptography.fernet import Fernet
import base64, hashlib
from app.config import settings


def _get_fernet() -> Fernet:
    """从配置密钥生成Fernet实例"""
    key = hashlib.sha256(settings.ENCRYPTION_KEY.encode()).digest()
    return Fernet(base64.urlsafe_b64encode(key))


def encrypt_api_key(plain_text: str) -> str:
    """加密API密钥"""
    if not plain_text:
        return ""
    f = _get_fernet()
    return f.encrypt(plain_text.encode()).decode()


def decrypt_api_key(encrypted_text: str) -> str:
    """解密API密钥"""
    if not encrypted_text:
        return ""
    try:
        f = _get_fernet()
        return f.decrypt(encrypted_text.encode()).decode()
    except Exception:
        return "[解密失败]"
