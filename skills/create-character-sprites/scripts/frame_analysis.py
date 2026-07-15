"""Shared alpha, matte, and body-anchor helpers for sprite scripts."""

from __future__ import annotations

import math
import statistics

from PIL import Image


def magenta_distance(red: int, green: int, blue: int) -> float:
    return math.sqrt((255 - red) ** 2 + green**2 + (255 - blue) ** 2)


def visible_magenta_pixels(image: Image.Image, distance_limit: int, alpha_limit: int) -> int:
    rgba = image.convert("RGBA")
    data = rgba.get_flattened_data() if hasattr(rgba, "get_flattened_data") else rgba.getdata()
    return sum(
        alpha >= alpha_limit and magenta_distance(red, green, blue) < distance_limit
        for red, green, blue, alpha in data
    )


def body_anchor(
    image: Image.Image,
    body_left: float = 0.30,
    body_right: float = 0.70,
    alpha_limit: int = 16,
) -> tuple[float, int]:
    """Return the foot-center proxy and baseline while ignoring outer props."""
    if not 0 <= body_left < body_right <= 1:
        raise ValueError("body bounds must satisfy 0 <= left < right <= 1")
    alpha = image.convert("RGBA").getchannel("A")
    left = round(alpha.width * body_left)
    right = round(alpha.width * body_right)
    for baseline in range(alpha.height - 1, -1, -1):
        if any(alpha.getpixel((x, baseline)) >= alpha_limit for x in range(left, right)):
            break
    else:
        raise ValueError("no visible body pixels in the selected body range")

    foot_pixels = [
        x
        for y in range(max(0, baseline - 3), baseline + 1)
        for x in range(left, right)
        if alpha.getpixel((x, y)) >= alpha_limit
    ]
    if not foot_pixels:
        raise ValueError("no visible foot pixels near the detected baseline")
    return float(statistics.median(foot_pixels)), baseline
