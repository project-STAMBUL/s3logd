from minio import Minio

from .logging_utils import get_logger

logger = get_logger(__name__)


def get_client(endpoint: str, access_key: str, secret_key: str, retry_num: int = 5):
    _client = None
    for retry_i in range(retry_num):
        try:
            _client = Minio(
                endpoint,
                secure=False,
                access_key=access_key,
                secret_key=secret_key,
            )
            break
        except Exception as e:
            logger.error("Exception while getting client:", e)
    return _client
