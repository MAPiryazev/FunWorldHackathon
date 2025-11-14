from pydantic import BaseModel, Field
from typing import Optional, List, Dict, Any
from enum import Enum

class MessageType(str, Enum):
    TEXT = "text"
    IMAGE = "image"
    DOCUMENT = "document"

class MaxMessage(BaseModel):
    """Универсальная модель входящего сообщения от пользователя."""
    message_id: str
    user_id: str
    chat_id: str
    text: str
    timestamp: int
    message_type: MessageType = MessageType.TEXT
    attachments: List[Dict[str, Any]] = Field(default_factory=list)
    username: Optional[str] = None
    first_name: Optional[str] = None

class BotResponse(BaseModel):
    """Модель ответа бота для отправки пользователю."""
    chat_id: str
    text: str
    reply_to_message_id: Optional[int] = None
    parse_mode: Optional[str] = "Markdown"
    user_id: Optional[str] = None