# EasyCAT known-bad overlap fixture

This directory stores the 192-byte EasyCAT project and generated artifacts that keep EasyCAT's default input SyncManager address `0x1200`.

EasyCAT writes generated artifacts by keeping the project basename and changing only the suffix:

- `device.yaml`
- `known_bad_overlap.prj`
- `known_bad_overlap.xml`
- `known_bad_overlap.bin`
- `known_bad_overlap.h`

The XML is intentionally unsafe for explicit triple-buffer validation: output process data starts at `0x1000`, has length `192`, and occupies `0x1000 + 3 * 192 = 0x1240`. The input SyncManager starts at `0x1200`, so the triple-buffer output span overlaps the input span by 64 bytes.

Keep this fixture as evidence for the overlap regression. Passing 192-byte references belong in `fixtures/easycat/ethercat_configurator_reference` with the `ethercat_configurator_reference.*` basename.