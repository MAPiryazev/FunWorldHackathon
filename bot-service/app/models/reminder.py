from pydantic import BaseModel
from typing import Optional
from enum import Enum

class ReminderPriority(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

class ReminderRequest(BaseModel):
    user_id: str
    text: str
    remind_at: str  # RFC3339 format
    complexity: int
    priority: Optional[str] = "medium"
    category: Optional[str] = "другие"
    notes: Optional[str] = None