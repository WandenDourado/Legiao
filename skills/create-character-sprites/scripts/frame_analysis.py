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


def foreground_bbox(image: Image.Image, alpha_limit: int = 16) -> tuple[int, int, int, int]:
    """Return the opaque-foreground bounds, ignoring near-transparent matte residue."""
    if not 0 < alpha_limit <= 255:
        raise ValueError("alpha_limit must be between 1 and 255")
    alpha = image.convert("RGBA").getchannel("A")
    mask = alpha.point(lambda value: 255 if value >= alpha_limit else 0)
    bbox = mask.getbbox()
    if bbox is None:
        raise ValueError("no visible foreground pixels")
    return bbox


def torso_center(
    image: Image.Image,
    top: float = 0.25,
    bottom: float = 0.65,
    alpha_limit: int = 16,
) -> float:
    """Return a stable horizontal pivot from the opaque torso band."""
    if not 0 <= top < bottom <= 1:
        raise ValueError("torso bounds must satisfy 0 <= top < bottom <= 1")
    alpha = image.convert("RGBA").getchannel("A")
    start = round(alpha.height * top)
    end = round(alpha.height * bottom)
    pixels = [
        x
        for y in range(start, end)
        for x in range(alpha.width)
        if alpha.getpixel((x, y)) >= alpha_limit
    ]
    if not pixels:
        raise ValueError("no visible torso pixels in the selected band")
    return float(statistics.median(pixels))


def nontransparent_border_pixels(image: Image.Image, border: int, alpha_limit: int) -> int:
    """Count border pixels that are too opaque for a clean transparent frame edge."""
    if border < 1 or border * 2 > min(image.size):
        raise ValueError("border must fit inside the image")
    alpha = image.convert("RGBA").getchannel("A")
    count = 0
    for y in range(alpha.height):
        for x in range(alpha.width):
            if x < border or y < border or x >= alpha.width - border or y >= alpha.height - border:
                if alpha.getpixel((x, y)) > alpha_limit:
                    count += 1
    return count


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
