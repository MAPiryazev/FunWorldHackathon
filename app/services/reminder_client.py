import httpx
import logging
from typing import Optional, Dict, Any, List
from app.config import settings

logger = logging.getLogger(__name__)

class ReminderClient:

    def __init__(self):
        self.base_url = settings.REMINDER_SERVICE_URL
        self.logger = logger

    async def create_task(self, task_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                self.logger.info(f"Создание задачи в reminder-service: {task_data}")

                resp = await client.post(
                    f"{self.base_url}/tasks",
                    json=task_data
                )

                if resp.status_code == 201:
                    result = resp.json()
                    self.logger.info(f"Задача создана в reminder-service: {result.get('id')}")
                    return result
                else:
                    self.logger.error(f"Ошибка reminder-service: {resp.status_code} - {resp.text}")
                    return None

        except httpx.ConnectError:
            self.logger.error("Не удалось подключиться к reminder-service")
            return None
        except Exception as e:
            self.logger.error(f"Ошибка при запросе к reminder-service: {e}")
            return None


    async def cancel_task(self, task_id: str) -> bool:
        """Отмена задачи в reminder-service"""
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.post(
                    f"{self.base_url}/tasks/{task_id}/cancel"
                )
                return resp.status_code == 200
        except Exception as e:
            self.logger.error(f"Ошибка отмены задачи: {e}")
            return False

    async def get_user_tasks(self, user_id: str, status: str = None, category: str = None) -> Optional[List[Dict[str, Any]]]:
        """Получение задач пользователя с фильтрацией"""
        try:
            async with httpx.AsyncClient() as client:
                params = {"user_id": user_id}
                if status:
                    params["status"] = status
                if category:
                    params["category"] = category

                self.logger.info(f"Запрос задач пользователя {user_id}")

                resp = await client.get(
                    f"{self.base_url}/tasks",
                    params=params
                )

                if resp.status_code == 200:
                    tasks = resp.json()
                    self.logger.info(f"Получено {len(tasks)} задач для пользователя {user_id}")
                    return tasks
                else:
                    self.logger.error(f"Ошибка получения задач: {resp.status_code}")
                    return None

        except httpx.ConnectError:
            self.logger.error("Не удалось подключиться к reminder-service")
            return None
        except Exception as e:
            self.logger.error(f"Ошибка получения задач: {e}")
            return None