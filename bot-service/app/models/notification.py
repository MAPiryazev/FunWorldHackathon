from pydantic import BaseModel
from typing import Optional


class NotificationRequest(BaseModel):
    id: str
    user_id: str
    text: str
    remind_at: str
    status: str
    retry_count: int
    created_at: str
    updated_at: str
    complexity: int
    priority: str
    category: str
    notes: Optional[str] = None
    chat_id: Optional[str] = None  # ID чата для отправки уведомления (MAX Messenger)