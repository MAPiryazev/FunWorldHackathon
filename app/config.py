from pathlib import Path
from typing import Optional

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Глобальные настройки сервиса."""

    # MAX Messenger
    MAX_BOT_TOKEN: str
    MAX_WEBHOOK_SECRET: Optional[str] = None

    # Интеграции
    NLP_SERVICE_URL: str = "http://localhost:8000"
    REMINDER_SERVICE_URL: str = "http://localhost:8907"

    # Веб-приложение
    BOT_HOST: str = "0.0.0.0"
    BOT_PORT: int = 8081
    WEBHOOK_PATH: str = "/max/webhook"
    DEBUG: bool = False

    # Tuna tunnel (опционально автозапуск)
    TUNA_AUTOSTART: bool = True
    TUNA_BIN: str = "tuna"
    TUNA_TOKEN: Optional[str] = None
    TUNA_DOMAIN: Optional[str] = None
    TUNA_LOCATION: Optional[str] = None
    TUNA_PUBLIC_URL: Optional[str] = None  # fallback при ручном запуске

    model_config = SettingsConfigDict(
        env_file=Path(__file__).parent.parent / ".env",
        env_file_encoding="utf-8-sig",
        case_sensitive=False,
        extra="ignore",
    )


settings = Settings()