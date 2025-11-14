from datetime import datetime
from typing import Optional, Dict
from app.models.user import User, UserState


class StateManager:
    """Менеджер состояния диалога пользователя"""

    def __init__(self):
        self._user_states: Dict[str, User] = {}

    async def get_user_state(self, user_id: str) -> UserState:
        """Получение состояния пользователя"""
        if user_id in self._user_states:
            return self._user_states[user_id].state
        return UserState.IDLE

    async def set_user_state(self, user_id: str, state: UserState, context: Optional[dict] = None):
        """Установка состояния пользователя"""
        current_time = datetime.now()

        if user_id in self._user_states:
            self._user_states[user_id].state = state
            self._user_states[user_id].last_activity_at = current_time
            if context:
                self._user_states[user_id].context = context
        else:
            self._user_states[user_id] = User(
                user_id=user_id,
                state=state,
                context=context,
                created_at=current_time,
                last_activity_at=current_time
            )

    async def get_user_context(self, user_id: str) -> Optional[dict]:
        """Получение контекста пользователя"""
        if user_id in self._user_states:
            return self._user_states[user_id].context
        return None

    async def clear_state(self, user_id: str):
        """Очистка состояния пользователя"""
        if user_id in self._user_states:
            self._user_states[user_id].state = UserState.IDLE
            self._user_states[user_id].context = None
            self._user_states[user_id].last_activity_at = datetime.utcnow()