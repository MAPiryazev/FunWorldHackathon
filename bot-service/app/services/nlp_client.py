import httpx
import logging
from typing import Optional, Dict, Any
from app.config import settings

logger = logging.getLogger(__name__)

class NLPClient:

    def __init__(self):
        self.base_url = settings.NLP_SERVICE_URL
        self.logger = logger

    async def parse_text(self, text: str, user_id: str) -> Optional[Dict[str, Any]]:

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                payload = {
                    'text': text,
                    'user_id': user_id,
                }

                self.logger.info("Отправка текста в nlp сервис")

                resp = await client.post(
                    f"{self.base_url}/parse",
                    json=payload
                )

                if resp.status_code == 200:
                    result = resp.json()
                    self.logger.info(f"nlp обработал текст: {result}")
                    return result
                else:
                    self.logger.error("Ошибка nlp сервиса")

                    return None
        except httpx.ConnectError:
            self.logger.error(" Не удалось подключиться к NLP сервису")
            return None
        except Exception as e:
            self.logger.error(f"Ошибка при запросе к NLP сервису: {e}")
            return None