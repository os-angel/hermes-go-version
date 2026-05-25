---
name: resumir-conversacion
description: Guia para resumir conversaciones largas y actualizar la memoria del agente
version: "1.0"
enabled: true
platforms:
  - whatsapp
  - email
  - restapi
prereqs:
  env: []
  files: []
---

# Skill: Resumir Conversacion

Este skill define como el agente resume conversaciones largas para mantener contexto sin saturar la ventana de tokens.

## Cuando usar este skill

Activa esta logica cuando:
- El usuario lleva mas de 20 turnos en la misma sesion.
- El usuario pide explicitamente un resumen: "resume lo que hablamos", "de que hemos hablado?".
- El contexto de la conversacion se volvio demasiado largo para una respuesta coherente.

## Procedimiento de resumen

1. Identifica los temas principales discutidos.
2. Extrae decisiones o compromisos importantes.
3. Identifica informacion relevante del usuario que deba persistir.
4. Escribe un resumen en 3-5 puntos concisos.

## Persistir resumen en memoria

Si el resumen contiene informacion relevante a largo plazo sobre el usuario o sus preferencias:

1. Usa `memory` con `action: "add"` y `target: "memory"` para guardar el contexto clave.
2. Usa `memory` con `action: "add"` y `target: "user"` para guardar datos del perfil del usuario.

No guardes el resumen completo de la conversacion; solo lo que sea util en sesiones futuras.

## Formato de respuesta al usuario

> "Hasta ahora hemos hablado de:
> - [punto 1]
> - [punto 2]
> - [punto 3]
>
> Continuo desde aqui."
