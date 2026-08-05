# customize plugins

`Customizer` interface — run post-mount customization. Errors become warnings.

| Key | Package | Description |
|-----|---------|-------------|
| `native` | native/ | Direct file injection/firstboot without external binary |
| `dynamicscripts` | dynamicscripts/ | Run external customization scripts |

Firstboot handlers for `dynamicscripts`: [`firstboot/plugins/README.md`](../firstboot/plugins/README.md).
