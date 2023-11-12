import logging
from typing import Optional


def get_logger(name: Optional[str] = None):
    logging.basicConfig(
        format="%(asctime)s:%(name)s:%(levelname)s:%(message)s",
        datefmt="%d/%m/%Y %H:%M:%S",
        level=logging.INFO,
    )
    logger = logging.getLogger(name)
    return logger
