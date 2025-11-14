import asyncio
import contextlib
import logging
import re
import shutil
from asyncio.subprocess import PIPE
from typing import Optional

from app.config import settings

logger = logging.getLogger(__name__)
_URL_PATTERN = re.compile(r"https?://[^\s,\"']+")


class TunaTunnel:
    """
    Простой менеджер туннеля через CLI tuna.

    Запускает команду `tuna http <port>` и пытается извлечь публичный URL из логов.
    При остановке приложения завершает запущенный процесс.
    """

    def __init__(self) -> None:
        self.process: Optional[asyncio.subprocess.Process] = None
        self.public_url: Optional[str] = settings.TUNA_PUBLIC_URL
        self._reader_task: Optional[asyncio.Task] = None
        self._detected_url: Optional[str] = None

    async def start(self) -> Optional[str]:
        """Запускает туннель и возвращает публичный URL (если удалось получить)."""
        if self.public_url:
            logger.info("Используем вручную заданный URL туннеля Tuna: %s", self.public_url)
            return self.public_url

        if not settings.TUNA_AUTOSTART:
            logger.info("Автозапуск туннеля отключен. Укажите TUNA_PUBLIC_URL для вебхука.")
            return None

        tuna_bin = shutil.which(settings.TUNA_BIN)
        if tuna_bin is None:
            logger.warning(
                "Команда '%s' не найдена. Установите Tuna CLI или задайте TUNA_PUBLIC_URL.",
                settings.TUNA_BIN,
            )
            return None

        args = [
            tuna_bin,
            "http",
            str(settings.BOT_PORT),
        ]

        if settings.TUNA_TOKEN:
            args.extend(["--token", settings.TUNA_TOKEN])
        if settings.TUNA_SUBDOMAIN:
            args.extend(["--subdomain", settings.TUNA_SUBDOMAIN])
        if settings.TUNA_DOMAIN:
            args.extend(["--domain", settings.TUNA_DOMAIN])
        if settings.TUNA_LOCATION:
            args.extend(["--location", settings.TUNA_LOCATION])

        logger.info("Запуск Tuna tunnel: %s", " ".join(args))
        self._detected_url = None
        self.process = await asyncio.create_subprocess_exec(
            *args,
            stdout=PIPE,
            stderr=PIPE,
        )
        self._reader_task = asyncio.create_task(self._consume_streams())
        await self._wait_for_url(timeout=15)

        if self._detected_url:
            logger.info("Tuna tunnel запущен по адресу: %s", self._detected_url)
        else:
            logger.warning("Не удалось автоматически определить URL туннеля Tuna.")

        return self._detected_url

    async def stop(self) -> None:
        """Останавливает туннель и освобождает ресурсы."""
        if self.process and self.process.returncode is None:
            logger.info("Остановка туннеля Tuna")
            self.process.terminate()
            try:
                await asyncio.wait_for(self.process.wait(), timeout=5)
            except asyncio.TimeoutError:  # pragma: no cover - аварийный случай
                self.process.kill()
        if self._reader_task:
            self._reader_task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await self._reader_task
        self.process = None
        self._reader_task = None

    async def _consume_streams(self) -> None:
        tasks = []
        if self.process and self.process.stdout:
            tasks.append(self._log_stream(self.process.stdout, logging.INFO))
        if self.process and self.process.stderr:
            tasks.append(self._log_stream(self.process.stderr, logging.INFO))
        if tasks:
            await asyncio.gather(*tasks, return_exceptions=True)

    async def _log_stream(self, stream: asyncio.StreamReader, level: int) -> None:
        while True:
            line = await stream.readline()
            if not line:
                break
            text = line.decode("utf-8", errors="ignore").strip()
            if not text:
                continue
            text_lower = text.lower()
            if any(keyword in text_lower for keyword in ["error", "failed", "fatal", "panic"]):
                log_level = logging.ERROR
            else:
                log_level = logging.INFO
            logger.log(log_level, "[tuna] %s", text)
            match = _URL_PATTERN.search(text)
            if match:
                candidate = match.group(0)
                if self._detected_url is None:
                    self._detected_url = candidate
                else:
                    current = self._detected_url
                    prefer_candidate = candidate.startswith("https://") or "tuna.am" in candidate
                    current_is_local = current.startswith("http://127.0.0.1")
                    if prefer_candidate and (current_is_local or not current.startswith("https://")):
                        self._detected_url = candidate

    async def _wait_for_url(self, timeout: int) -> None:
        step = 0.5
        waited = 0.0
        while waited < timeout:
            if self._detected_url and self._detected_url.startswith("https://"):
                return
            if self.process and self.process.returncode is not None:
                logger.warning(
                    "Процесс Tuna завершился с кодом %s. "
                    "Убедитесь, что установлен Tuna CLI и выполнена авторизация (tuna login), "
                    "или задайте TUNA_PUBLIC_URL вручную.",
                    self.process.returncode,
                )
                return
            await asyncio.sleep(step)
            waited += step


    @property
    def webhook_base(self) -> Optional[str]:
        """Возвращает URL туннеля, если он определен."""
        return self._detected_url or self.public_url

