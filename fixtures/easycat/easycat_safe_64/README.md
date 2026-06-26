# EasyCAT safe 64-byte reference fixture

This paired fixture proves that EasyCAT's default input SyncManager address `0x1200` is safe when output process data is only 64 bytes and the device YAML explicitly selects `buffer_mode: triple`.

Triple-buffer calculation:

```text
output SM: 0x1000 + 3 * 64 = 0x10c0
input SM:  0x1200
```

Workflow:

1. Keep `device.yaml` paired with `easycat_safe_64.prj` in this directory.
2. Open `F:\ethercat2peripheral\ethercat-peripheral-board\protocol\EasyCAT\easycat_safe_64\easycat_safe_64.prj` in EasyCAT Configurator.
3. Generate `.xml`, `.bin`, and `.h`.
4. Copy the generated files here as:
   - `easycat_safe_64.xml`
   - `easycat_safe_64.bin`
   - `easycat_safe_64.h`
5. Run `go test ./...`.
