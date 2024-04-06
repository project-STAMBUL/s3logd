import io
import re
import os
import time
from glob import glob
from typing import Dict
from datetime import datetime

import zoneinfo

from src.utils.logging_utils import get_logger
from src.utils.get_client import get_client

MINIO_ENDPOINT = os.environ["S3_ENDPOINT"]
ACCESS_KEY = os.environ["S3_ACCESS_KEY_ID"]
SECRET_KEY = os.environ["S3_SECRET_ACCESS_KEY"]
S3_BUCKET = os.environ["S3_BUCKET"]
ZONE = zoneinfo.ZoneInfo("Europe/Moscow")


def backup_worker(
    file_path: str,
    regex_pattern: str,
    clear_after_backup: bool,
    backup_check_rate: int = 3600,
):
    logger = get_logger(f"Backup: {file_path}")
    pattern = re.compile(regex_pattern)

    _client = None
    while _client is None:
        _client = get_client(MINIO_ENDPOINT, ACCESS_KEY, SECRET_KEY)
        if _client is not None:
            break
        logger.warning("Can't establish MINIO connection: %s", MINIO_ENDPOINT)
        time.sleep(backup_check_rate)

    time.sleep(backup_check_rate)
    files_to_check = glob(file_path + "*")
    try:
        for file in files_to_check:
            # Check if the file matches the pattern
            if pattern.match(file):
                logger.info("Found match to backup: %s, regex=%s", file, regex_pattern)
                object_name = os.path.join(
                    file.rsplit(".", 1)[-1], os.path.basename(file)
                )
                with open(file, "rb") as fin:
                    file_bytes = fin.read()
                    _client.put_object(
                        S3_BUCKET,
                        object_name,
                        io.BytesIO(file_bytes),
                        length=len(file_bytes),
                    )
                logger.info("Pushed %s to %s:%s", file_path, "S3_BUCKET", object_name)
                if clear_after_backup:
                    os.remove(file)
                break
            else:
                logger.warning(
                    "backup file %s;regex=%s doesn't exists",
                    file_path,
                    regex_pattern,
                )
    except Exception as ex:
        logger.error("Error: %s", ex)


def stream_worker(file_path: str, push_rate: int):
    """
    Беспрерывно стримим файл, который подходит под описание
    :param file_path:
    :param push_rate:
    :return:
    """
    logger = get_logger(f"Stream: {file_path}")

    _client = None
    while _client is None:
        _client = get_client(MINIO_ENDPOINT, ACCESS_KEY, SECRET_KEY)
        if _client is not None:
            break
        logger.warning("Can't establish MINIO connection: %s", MINIO_ENDPOINT)
        time.sleep(push_rate)

    time.sleep(push_rate)
    if not os.path.exists(file_path):
        logger.warning("stream file %s doesn't exists", file_path)
        return

    try:
        # Generating object name
        now = datetime.now(ZONE)
        object_name = os.path.basename(file_path)
        object_name = os.path.join(
            now.date().strftime("%Y-%m-%d"),
            "_".join([now.strftime("%H"), object_name]),
        )
        with open(file_path, "rb") as fin:
            file_bytes = fin.read()
            _client.put_object(
                S3_BUCKET,
                object_name,
                io.BytesIO(file_bytes),
                length=len(file_bytes),
            )
        logger.info("Pushed %s to %s:%s", file_path, "S3_BUCKET", object_name)
    except Exception as ex:
        logger.error("Error: %s", ex)


def minio_worker(
    stream: Dict,
):
    """
    Пушим в MINIO
    Args:
        stream: Аргументы потока
                file - путь до файла
                type: monitor
                    pushRate - как часто пушим
                type: backup
                    regex_pattern: regex для матчинга файлов
                    clear_after_backup: удалить после бэкапа
    """
    file_path = stream["file"]
    monitoring_type = stream["type"]

    if monitoring_type == "stream":
        push_rate = stream["pushRate"]
        while True:
            stream_worker(file_path=file_path, push_rate=push_rate)
    elif monitoring_type == "backup":
        regex_pattern = stream["regex_pattern"]
        clear_after_backup = stream["clear_after_backup"]
        while True:
            backup_worker(
                file_path,
                regex_pattern=regex_pattern,
                clear_after_backup=clear_after_backup,
                backup_check_rate=60 * 60,
            )
    else:
        raise ValueError(f"unknown {monitoring_type=}; Must be 'stream' or 'backup'")
