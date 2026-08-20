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


def silhouette_mask(image: Image.Image, alpha_limit: int = 16):
    """Return a 1-bit foreground mask cropped to its own bounding box.

    Cropping to the bbox removes translation so masks can be compared by shape.
    """
    alpha = image.convert("RGBA").getchannel("A")
    mask = alpha.point(lambda value: 255 if value >= alpha_limit else 0).convert("1")
    bbox = mask.getbbox()
    if bbox is None:
        raise ValueError("no visible foreground pixels")
    return mask.crop(bbox)


def _resize_mask(mask, size):
    return mask.convert("L").resize(size, Image.Resampling.NEAREST).point(lambda v: 255 if v >= 128 else 0).convert("1")


def mask_iou(mask_a, mask_b, size: tuple[int, int] = (64, 96)) -> float:
    """Intersection-over-union of two shape masks, resampled to a common box."""
    a = _resize_mask(mask_a, size)
    b = _resize_mask(mask_b, size)
    a_data = list(a.getdata())
    b_data = list(b.getdata())
    inter = sum(1 for pa, pb in zip(a_data, b_data) if pa and pb)
    union = sum(1 for pa, pb in zip(a_data, b_data) if pa or pb)
    return inter / union if union else 0.0


def content_patch(
    image: Image.Image,
    size: tuple[int, int] = (64, 96),
    alpha_limit: int = 16,
    body_fraction: float = 1.0,
):
    """Return a normalized grayscale patch of the character, cropped to its bbox.

    Uses interior luminance (face, folds, emblems) rather than the outline, because
    a facing flip barely changes a symmetric silhouette but clearly moves interior
    detail. Transparent pixels are set to mid-grey so they do not bias correlation.

    body_fraction < 1 keeps only the top fraction of the silhouette (head/torso).
    This matters for facing detection: in a legitimate walk cycle the LEGS of the
    second half genuinely mirror the first half, so whole-body correlation would
    flag a correct cycle as flipped. The upper body carries the facing signal
    (face, hair, asymmetric torso detail) without that legitimate mirroring.
    """
    rgba = image.convert("RGBA")
    bbox = foreground_bbox(rgba, alpha_limit)
    if body_fraction < 1.0:
        cut = bbox[1] + max(1, round((bbox[3] - bbox[1]) * body_fraction))
        bbox = (bbox[0], bbox[1], bbox[2], cut)
    cropped = rgba.crop(bbox)
    grey = cropped.convert("L")
    alpha = cropped.getchannel("A")
    grey = Image.composite(grey, Image.new("L", cropped.size, 128), alpha.point(lambda v: 255 if v >= alpha_limit else 0))
    return grey.resize(size, Image.Resampling.BILINEAR)


def _ncc(patch_a, patch_b) -> float:
    a = list(patch_a.getdata())
    b = list(patch_b.getdata())
    n = len(a)
    mean_a = sum(a) / n
    mean_b = sum(b) / n
    num = sum((x - mean_a) * (y - mean_b) for x, y in zip(a, b))
    den_a = sum((x - mean_a) ** 2 for x in a) ** 0.5
    den_b = sum((y - mean_b) ** 2 for y in b) ** 0.5
    if den_a == 0 or den_b == 0:
        return 0.0
    return num / (den_a * den_b)


def mirror_content_bias(image: Image.Image, reference_patch, alpha_limit: int = 16) -> tuple[float, float]:
    """Return (ncc_same, ncc_mirrored) of an image's interior content vs a reference.

    A frame whose facing is horizontally flipped relative to the reference correlates
    more strongly when mirrored. Works on symmetric silhouettes (robes) where an
    outline-based test is blind. Deterministic detector for row-2 inversion.
    """
    patch = content_patch(image, reference_patch.size, alpha_limit)
    same = _ncc(patch, reference_patch)
    mirrored = _ncc(patch.transpose(Image.Transpose.FLIP_LEFT_RIGHT), reference_patch)
    return same, mirrored


def lower_body_offset(
    image: Image.Image,
    torso_top: float = 0.25,
    torso_bottom: float = 0.65,
    band_top: float = 0.86,
    alpha_limit: int = 16,
) -> tuple[float, float]:
    """Return (lateral_offset, signal_strength) of the lower-body mass vs torso center.

    A positive offset means the lower-body silhouette leans right of the torso
    pivot. signal_strength is the visible lower-body width as a fraction of frame
    width; a small value (long robe hiding the legs) means the lateral offset is
    not a reliable lead-foot cue and the caller should fall back to a visual check.
    """
    alpha = image.convert("RGBA").getchannel("A")
    width, height = alpha.size
    pivot = torso_center(image, torso_top, torso_bottom, alpha_limit)
    top = round(height * band_top)
    xs = [x for y in range(top, height) for x in range(width) if alpha.getpixel((x, y)) >= alpha_limit]
    if not xs:
        return 0.0, 0.0
    offset = statistics.median(xs) - pivot
    strength = (max(xs) - min(xs)) / width
    return offset, strength


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
