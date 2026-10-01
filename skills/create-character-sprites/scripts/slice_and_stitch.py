#!/usr/bin/env python3
"""Split a 2x4 AI grid into keyed RGBA frames for one direction."""

from __future__ import annotations

import argparse
from pathlib import Path

try:
    from PIL import Image
except ImportError as exc:  # pragma: no cover
    raise SystemExit("Pillow is required. Install with: python -m pip install pillow") from exc

from frame_analysis import magenta_distance


def matte_to_alpha(image: Image.Image, transparent_threshold: int, feather_threshold: int) -> Image.Image:
    """Key a magenta matte to true alpha, keeping a narrow despilled edge feather."""
    if not 0 < transparent_threshold < feather_threshold <= 255:
        raise ValueError("matte thresholds must satisfy 0 < transparent < feather <= 255")
    pixels = []
    rgba = image.convert("RGBA")
    data = rgba.get_flattened_data() if hasattr(rgba, "get_flattened_data") else rgba.getdata()
    for red, green, blue, alpha in data:
        distance = magenta_distance(red, green, blue)
        if distance <= transparent_threshold:
            pixels.append((0, 0, 0, 0))
            continue
        if distance < feather_threshold:
            feather = (distance - transparent_threshold) / (feather_threshold - transparent_threshold)
            alpha = round(alpha * feather)
            # True unmix, not green-boost despill: the pixel is c = f*fg + (1-f)*matte,
            # so recover fg = (c - (1-f)*M) / f with M=(255,0,255). Green-boost only
            # desaturated the blend, leaving a pink outline over real backgrounds.
            if feather > 0.12:
                spill = (1 - feather) * 255
                red = max(0, min(255, round((red - spill) / feather)))
                green = max(0, min(255, round(green / feather)))
                blue = max(0, min(255, round((blue - spill) / feather)))
            else:
                red, green, blue, alpha = 0, 0, 0, 0
        pixels.append((red, green, blue, alpha))
    result = Image.new("RGBA", image.size)
    result.putdata(pixels)
    return result


def clear_near_transparent_pixels(
    image: Image.Image,
    alpha_limit: int = 4,
    residual_alpha_limit: int = 64,
    residual_matte_threshold: int = 160,
) -> Image.Image:
    """Remove hidden and low-alpha magenta fringes after keying or resampling."""
    pixels = []
    data = image.convert("RGBA").get_flattened_data() if hasattr(image, "get_flattened_data") else image.convert("RGBA").getdata()
    for red, green, blue, alpha in data:
        residual_matte = alpha <= residual_alpha_limit and magenta_distance(red, green, blue) < residual_matte_threshold
        pixels.append((0, 0, 0, 0) if alpha <= alpha_limit or residual_matte else (red, green, blue, alpha))
    result = Image.new("RGBA", image.size)
    result.putdata(pixels)
    return result


def segment_figures(source: Image.Image, matte_threshold: int, downscale: int = 4, min_run_ratio: float = 0.02):
    """Return [(start_x, end_x), ...] figure column-runs found on the matte, full-res coords.

    Generators sometimes draw the wrong number of figures (7 instead of 4) or misalign
    them against the uniform grid; counting content runs on the matte catches both
    before any cell is cut through a body.
    """
    small = source.convert("RGB").resize((max(1, source.width // downscale), max(1, source.height // downscale)))
    width, height = small.size
    pixels = small.load()
    is_content = []
    for x in range(width):
        content = False
        for y in range(height):
            red, green, blue = pixels[x, y]
            if magenta_distance(red, green, blue) >= matte_threshold:
                content = True
                break
        is_content.append(content)
    runs = []
    start = None
    for x, content in enumerate(is_content + [False]):
        if content and start is None:
            start = x
        elif not content and start is not None:
            runs.append((start, x))
            start = None
    min_width = max(2, round(width * min_run_ratio))
    runs = [(a * downscale, b * downscale) for a, b in runs if (b - a) >= min_width]
    return runs


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, help="Unguttered 2x4 grid image.")
    parser.add_argument("--direction", required=True, help="Direction folder name, for example S.")
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--matte-threshold", type=int, default=120, help="Distance from magenta that becomes fully transparent.")
    parser.add_argument("--matte-feather-threshold", type=int, default=190, help="Distance from magenta that becomes fully opaque.")
    parser.add_argument("--minimum-source-scale", type=float, default=1.0)
    parser.add_argument("--grid-cols", type=int, default=4, help="Columns in this grid (4 for a full row).")
    parser.add_argument("--grid-rows", type=int, default=2, help="Rows in this grid (2 for a full 8-pose grid, 1 for a 4-pose batch).")
    parser.add_argument("--frame-offset", type=int, default=0, help="Index of the first output frame; use 4 for the second batch.")
    parser.add_argument("--trim-tolerance", type=int, default=8, help="Max remainder px per cell to auto-trim to a divisible grid before failing.")
    parser.add_argument("--aspect-tolerance", type=float, default=0.12, help="Allowed cell-aspect deviation from the target before the content-fit fallback engages.")
    parser.add_argument("--content-target-ratio", type=float, default=0.84, help="Silhouette height ratio used by the content-fit fallback.")
    parser.add_argument("--min-visible-alpha", type=int, default=4,
                        help="Post-key cleanup floor: pixels at or below this alpha are cleared. Use ~24 for supersampled (2x) work, where faint matte feather survives and inflates measurements.")
    parser.add_argument("--anchor-x", type=int, default=None,
                        help="Torso pivot used by the content-fit fallback (default: frame width / 2 — scales with 2x frames).")
    parser.add_argument("--baseline", type=int, default=None,
                        help="Foot baseline used by the content-fit fallback (default: 186 scaled to frame height).")
    parser.add_argument("--skip-figure-count", action="store_true",
                        help="Skip the matte figure-count verification (single-row grids only).")
    args = parser.parse_args()
    if args.anchor_x is None:
        args.anchor_x = args.frame_width // 2
    if args.baseline is None:
        args.baseline = round(186 * args.frame_height / 192)

    if args.frame_width <= 0 or args.frame_height <= 0 or not 0 < args.matte_threshold < args.matte_feather_threshold <= 255:
        raise SystemExit("Frame dimensions and matte thresholds must satisfy 0 < threshold < feather <= 255.")
    if args.minimum_source_scale < 1:
        raise SystemExit("--minimum-source-scale must be at least 1.")
    if args.grid_cols < 1 or args.grid_rows < 1 or args.frame_offset < 0:
        raise SystemExit("Grid columns/rows must be >= 1 and frame offset >= 0.")
    if not args.input.is_file():
        raise SystemExit(f"Input does not exist: {args.input}")

    with Image.open(args.input) as source:
        # Image generators rarely return exact multiples (e.g. 1983x793). Trim a few
        # remainder pixels to the nearest divisible size instead of forcing a costly
        # regeneration; refuse only if the remainder is too large to be rounding.
        rem_w, rem_h = source.width % args.grid_cols, source.height % args.grid_rows
        max_trim_w = args.grid_cols * args.trim_tolerance
        max_trim_h = args.grid_rows * args.trim_tolerance
        if rem_w > max_trim_w or rem_h > max_trim_h:
            raise SystemExit(
                f"Grid {source.width}x{source.height} is not close to a {args.grid_cols}x{args.grid_rows} multiple "
                f"(remainder {rem_w}x{rem_h} exceeds tolerance); regenerate at the requested dimensions."
            )
        if rem_w or rem_h:
            left, top = rem_w // 2, rem_h // 2
            source = source.crop((left, top, source.width - (rem_w - left), source.height - (rem_h - top)))
            print(f"Note: trimmed {rem_w}x{rem_h} remainder px to reach a divisible {source.width}x{source.height} grid.")
        crop_width, crop_height = source.width // args.grid_cols, source.height // args.grid_rows
        # Generators often ignore the exact aspect ratio. Rather than distort (a hard
        # resize to the target frame) or fail, fit each cell into the frame preserving
        # aspect with transparent padding. A wildly-off cell aspect (e.g. a portrait
        # canvas with huge matte margins) triggers the content-fit fallback below
        # instead of a costly regeneration, as long as the cell ORDER is correct.
        target_ratio = args.frame_width / args.frame_height
        cell_ratio = crop_width / crop_height
        content_fit = not (1 - args.aspect_tolerance) <= cell_ratio / target_ratio <= (1 + args.aspect_tolerance)
        if content_fit:
            print(
                f"Note: cell aspect {cell_ratio:.3f} is far from target {target_ratio:.3f}; "
                f"using content-fit fallback (crop to figure, common scale, re-anchor)."
            )
        source_scale = min(crop_width / args.frame_width, crop_height / args.frame_height)
        if source_scale < args.minimum_source_scale:
            raise SystemExit(
                f"Grid cells are only {source_scale:.2f}x the target frame; expected at least "
                f"{args.minimum_source_scale:.2f}x for the requested export quality."
            )
        source = source.convert("RGBA")
        if args.grid_rows == 1 and not args.skip_figure_count:
            runs = segment_figures(source, args.matte_threshold)
            expected = args.grid_cols
            if len(runs) != expected:
                raise SystemExit(
                    f"Figure count mismatch: found {len(runs)} figures on the matte, expected exactly {expected}. "
                    f"Regenerate this batch instructing EXACTLY {expected} evenly-spaced figures, one per cell "
                    f"(or figures are touching/merged - add spacing)."
                )
            centers = [(a + b) // 2 for a, b in runs]
            uniform_centers = [column * crop_width + crop_width // 2 for column in range(expected)]
            misaligned = any(abs(c - u) > crop_width * 0.10 for c, u in zip(centers, uniform_centers))
            if misaligned:
                print("Note: figures are misaligned with the uniform grid; recentering cells on figure centers.")
                frames = []
                for center in centers:
                    left = min(max(0, center - crop_width // 2), source.width - crop_width)
                    frames.append(source.crop((left, 0, left + crop_width, crop_height)))
            else:
                frames = [source.crop((column * crop_width, 0, (column + 1) * crop_width, crop_height)) for column in range(expected)]
        else:
            frames = [
                source.crop((column * crop_width, row * crop_height, (column + 1) * crop_width, (row + 1) * crop_height))
                for row in range(args.grid_rows)
                for column in range(args.grid_cols)
            ]

    output_dir = args.output_root / args.direction.strip().upper()
    output_dir.mkdir(parents=True, exist_ok=True)
    resampling = Image.Resampling.LANCZOS
    keyed = [
        clear_near_transparent_pixels(
            matte_to_alpha(frame, args.matte_threshold, args.matte_feather_threshold),
            alpha_limit=args.min_visible_alpha,
        )
        for frame in frames
    ]
    if content_fit:
        # Content-fit: crop each keyed cell to its figure, scale ALL cells by one
        # common factor (median height -> target ratio) so relative pose heights are
        # preserved, and anchor each on the fixed pivot/baseline.
        import statistics

        from frame_analysis import body_anchor, foreground_bbox, torso_center

        boxes = [foreground_bbox(cell, 16) for cell in keyed]
        heights = [box[3] - box[1] for box in boxes]
        # Match the scale of frames already sliced into this direction (e.g. batch 1
        # when slicing batch 2), so the two halves never end up at different sizes.
        target_px = args.content_target_ratio * args.frame_height
        existing = sorted((args.output_root / args.direction.strip().upper()).glob("*.png"))
        existing = [p for p in existing if int(p.stem) not in range(args.frame_offset, args.frame_offset + len(keyed))]
        if existing:
            sibling_heights = []
            for path in existing:
                with Image.open(path) as sib:
                    box = foreground_bbox(sib.convert("RGBA"), 16)
                    sibling_heights.append(box[3] - box[1])
            target_px = statistics.median(sibling_heights)
            print(f"Note: content-fit matching existing sibling frames' scale ({target_px:.0f}px height).")
        factor = target_px / statistics.median(heights)
        fitted_cells = []
        for cell, box in zip(keyed, boxes, strict=True):
            cropped = cell.crop(box)
            scaled = cropped.resize(
                (max(1, round(cropped.width * factor)), max(1, round(cropped.height * factor))), resampling
            )
            scaled = clear_near_transparent_pixels(scaled, alpha_limit=args.min_visible_alpha)
            tx = torso_center(scaled)
            _, by = body_anchor(scaled)
            offset_xy = (round(args.anchor_x - tx), args.baseline - by)
            bbox = foreground_bbox(scaled, 16)
            if (bbox[0] + offset_xy[0] < 0 or bbox[2] + offset_xy[0] > args.frame_width
                    or bbox[1] + offset_xy[1] < 0 or bbox[3] + offset_xy[1] > args.frame_height):
                raise SystemExit(
                    "content-fit: a figure does not fit the frame at the common scale/anchor "
                    "(too wide or off-center); regenerate this batch centered with side margins."
                )
            canvas = Image.new("RGBA", (args.frame_width, args.frame_height))
            canvas.alpha_composite(scaled, offset_xy)
            fitted_cells.append(clear_near_transparent_pixels(canvas, alpha_limit=args.min_visible_alpha))
        keyed = fitted_cells
    for offset, frame in enumerate(keyed):
        index = args.frame_offset + offset
        if not content_fit:
            # Aspect-preserving fit into the target frame (identical to an exact resize
            # when the cell already matches the frame aspect; pads transparently
            # otherwise, never distorting the character).
            scale = min(args.frame_width / frame.width, args.frame_height / frame.height)
            fitted_size = (max(1, round(frame.width * scale)), max(1, round(frame.height * scale)))
            fitted = frame.resize(fitted_size, resampling)
            canvas = Image.new("RGBA", (args.frame_width, args.frame_height))
            canvas.alpha_composite(fitted, ((args.frame_width - fitted_size[0]) // 2, (args.frame_height - fitted_size[1]) // 2))
            frame = clear_near_transparent_pixels(canvas, alpha_limit=args.min_visible_alpha)
        frame.save(output_dir / f"{index:03d}.png")

    print(f"OK: wrote {len(frames)} RGBA frames to {output_dir} (indices {args.frame_offset:03d}-{args.frame_offset + len(frames) - 1:03d}) from {source_scale:.2f}x source cells")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
