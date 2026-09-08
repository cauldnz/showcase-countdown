"""Produce a single flashable image for browser-based installers.

ESP Web Tools cannot patch flash mode, frequency, and size on the fly the way
esptool does when writing the four separate ESP32 images, so it needs one blob
starting at offset 0. This runs after every successful link and writes
`merged-firmware.bin` next to the normal build output.
"""

import os

Import("env")  # noqa: F821

BOOTLOADER_OFFSET = 0x1000
PARTITIONS_OFFSET = 0x8000
BOOT_APP0_OFFSET = 0xE000
APPLICATION_OFFSET = 0x10000


def _boot_app0_path(env):
    packages = env.subst("$PROJECT_PACKAGES_DIR")
    return os.path.join(
        packages, "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"
    )


def merge_firmware(source, target, env):
    build_dir = env.subst("$BUILD_DIR")
    board = env.BoardConfig()

    parts = [
        (BOOTLOADER_OFFSET, os.path.join(build_dir, "bootloader.bin")),
        (PARTITIONS_OFFSET, os.path.join(build_dir, "partitions.bin")),
        (BOOT_APP0_OFFSET, _boot_app0_path(env)),
        (APPLICATION_OFFSET, os.path.join(build_dir, "firmware.bin")),
    ]

    missing = [path for _, path in parts if not os.path.isfile(path)]
    if missing:
        print("merge_firmware: skipped, missing %s" % ", ".join(missing))
        return

    output = os.path.join(build_dir, "merged-firmware.bin")

    # ESP Web Tools requires dio; qio images do not boot once written verbatim.
    flash_mode = board.get("build.flash_mode", "dio")
    if flash_mode in ("qio", "qout"):
        flash_mode = "dio"
    elif flash_mode in ("opi_opi", "opi_qspi"):
        flash_mode = "dout"

    command = [
        "$PYTHONEXE",
        '"%s"' % env.subst("$OBJCOPY"),
        "--chip",
        board.get("build.mcu", "esp32"),
        "merge_bin",
        "-o",
        '"%s"' % output,
        "--flash_mode",
        flash_mode,
        "--flash_freq",
        board.get("build.f_flash", "40000000L").replace("000000L", "m"),
        "--flash_size",
        board.get("upload.flash_size", "4MB"),
    ]
    for offset, path in parts:
        command.extend([hex(offset), '"%s"' % path])

    env.Execute(" ".join(command))
    print("merge_firmware: wrote %s" % output)


env.AddPostAction("$BUILD_DIR/${PROGNAME}.bin", merge_firmware)  # noqa: F821
