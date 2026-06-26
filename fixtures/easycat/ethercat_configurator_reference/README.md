# EasyCAT reference fixture

This directory stores the paired EasyCAT project and generated passing reference artifacts for `examples/easycat-reference/device.yaml`.

This is the 192-byte case. EasyCAT's default input SM address `0x1200` is intentionally tracked as a known-bad artifact because it overlaps with the output SM when the paired device YAML explicitly selects `buffer_mode: triple`. See `fixtures/easycat/easycat_safe_64` for a smaller 64-byte case where the same default gap is safe.

Workflow:

1. Open `F:\ethercat2peripheral\ethercat-peripheral-board\protocol\EasyCAT\ethercat_configurator_reference\ethercat_configurator_reference.prj` in EasyCAT Configurator.
2. Generate the EasyCAT `.xml`, `.bin`, and `.h` files with the GUI.
3. For a passing reference, ensure the generated or patched XML/SII use input SyncManager address `0x1800`. With 192-byte process data and explicit triple-buffer SyncManagers, `0x1000 + 3 * 192 = 0x1240`, so an input SM at `0x1200` overlaps the output SM by 64 bytes.
4. Copy the fixed reference files into this directory with these names:
   - `ethercat_configurator_reference.xml`
   - `ethercat_configurator_reference.bin`
   - `ethercat_configurator_reference.h`
5. Run `go test ./...`.

The reference tests focus on XML and binary identity/PDI/PDO compatibility. Header output is intentionally not compared byte-for-byte.
