# Watcher Service

Google Calendar API provides limited capabilities when it comes to fully automating calendar synchronization. While it supports event updates, push notifications (watch), and incremental sync via sync tokens, it does not handle the complete synchronization logic for you.

**Watcher Service** bridges this gap by:

- **Registering webhook**s for each Google-authorized user’s calendar to detect new or updated events.
- **Forwarding changes** to the backend for processing.
- Allowing the backend to decide whether to replicate, synchronize, or ignore changes.

This design enables incremental and targeted synchronization while handling recurring events, conflict resolution, and other complex scenarios that the Calendar API does not automate out of the box.
