---
name: gestionar-tareas
description: Guia para ayudar al usuario a gestionar sus tareas y recordatorios
version: "1.0"
enabled: true
platforms:
  - whatsapp
  - restapi
prereqs:
  env: []
  files: []
---

# Skill: Gestionar Tareas

Este skill define como el agente ayuda al usuario a registrar, recordar y hacer seguimiento de sus tareas.

## Crear una tarea

Cuando el usuario pida recordar algo o crear una tarea:

1. Extrae el contenido de la tarea de forma clara.
2. Confirma con el usuario si hay fecha o prioridad.
3. Usa la tool `memory` con `action: "add"` y `target: "memory"` para persistir la tarea.
4. Confirma al usuario que la tarea fue guardada.

Formato recomendado en memoria:
```
TAREA: [descripcion]
Fecha: [fecha si aplica]
Estado: pendiente
```

## Listar tareas

Cuando el usuario pregunte por sus tareas pendientes:

1. Usa `memory` con `action: "read"` y `target: "memory"`.
2. Filtra las entradas que empiecen con "TAREA:".
3. Presenta la lista de forma clara y ordenada.

## Completar o eliminar una tarea

Cuando el usuario marque una tarea como completa:

1. Usa `memory` con `action: "replace"` para cambiar "Estado: pendiente" a "Estado: completado".
2. Confirma al usuario.

## Recordatorios programados

Para recordatorios recurrentes, sugiere al usuario usar el sistema de cron del agente:

> "Puedo programar un recordatorio automatico. Indicame la frecuencia (ej. cada lunes a las 9am)."
