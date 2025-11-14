from pydantic import BaseModel
from datetime import datetime
from typing import Optional
from enum import Enum

class Priority(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

class Category(str, Enum):
    WORK = "работа"
    PERSONAL = "личное"
    HEALTH = "здоровье"
    SHOPPING = "покупки"
    OTHER = "другие"

# Для запросов
class ParseRequest(BaseModel):
    text: str
    user_id: str


# Для внутреннего использования в парсере
class ExtractedData(BaseModel):
    text: str
    remind_at: Optional[datetime] = None
    priority: Priority = Priority.MEDIUM
    category: Category = Category.OTHER
    complexity: int = 1
    notes: Optional[str] = None


# Для ответов
class ParseResponse(BaseModel):
    user_id: str                    # кто создал задачу
    text: str                       # текст задачи / уведомления
    remind_at: Optional[datetime] = None  # время для отправки
    complexity: int = 1             # 1–5 (сложность)
    priority: str = "medium"        # low | medium | high
    category: str = "другие"        # работа | личное и т.п.
    notes: Optional[str] = None     # дополнительные заметки


# Автоматически преобразуем Enum в строки
class Config:
    use_enum_values = True