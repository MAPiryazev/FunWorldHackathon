from datetime import datetime
from pydantic import BaseModel
from typing import Optional, Dict, Any
from enum import Enum

class UserState(str, Enum):
    IDLE = "idle"
    AWAITING_TIME = "awaiting_time"

class User(BaseModel):
    user_id: str
    username: Optional[str] = None
    state: UserState = UserState.IDLE
    context: Optional[Dict[str, Any]] = None
    created_at: Optional[datetime] = None
    last_activity_at: Optional[datetime] = None