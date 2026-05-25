# bridge

HTTP bridge Node.js que conecta hermes-go con WhatsApp via Baileys.

## Instalacion

```bash
cd bridge
npm install
```

## Uso directo

```bash
node bridge.js --port 3001 --session /ruta/a/session --mode bot
```

## Argumentos

| Argumento  | Default           | Descripcion                        |
|------------|-------------------|------------------------------------|
| `--port`   | `3001`            | Puerto HTTP                        |
| `--session`| `./session`       | Directorio para auth_state Baileys |
| `--mode`   | `bot`             | `bot` o `user`                     |

## Endpoints

| Metodo | Ruta           | Descripcion                                        |
|--------|----------------|----------------------------------------------------|
| GET    | `/health`      | Health check. Retorna `{ status: "ok" }`           |
| GET    | `/status`      | Estado de conexion. `{ connected, phone }`         |
| GET    | `/messages`    | Retorna y vacia la cola de mensajes pendientes     |
| POST   | `/send`        | Envia texto. Body: `{ chatId, message, replyTo? }` |
| POST   | `/send-media`  | Envia archivo. Body: `{ chatId, filePath, ... }`   |
| POST   | `/typing`      | Envia indicador de escritura. Body: `{ chatId }`   |

## Primer uso (autenticacion QR)

1. Arrancar el bridge: `node bridge.js --session ./session`
2. Escanear el QR impreso en terminal con WhatsApp (Dispositivos vinculados)
3. Las credenciales quedan guardadas en `--session`, no hace falta re-escanear

## Notas

- hermes-go arranca el bridge automaticamente como subproceso.
- Los logs del bridge van a `~/.hermes-go/logs/bridge-stderr.log`.
- El bridge escucha solo en `127.0.0.1` (no expuesto externamente).
