# Roadmap

## v0.2.0 ✅ (current)

- [x] Custom CORS middleware (reemplazar gin-contrib/cors)
- [x] Remote definitions desde Worker con fallback local
- [x] Sistema de notificación de versión (campanita)
- [x] API endpoints `/api/version` y `/api/definitions`
- [x] Landing con descarga dinámica
- [x] Inyección de versión via ldflags

## v0.3.0 (next)

- [ ] **Auto-update**: el backend descarga la nueva versión y la instala automáticamente
- [ ] **Configuración de update channel**: stable / beta / dev
- [ ] **Métrica de uso anónima**: conteo de dispositivos conectados (opt-in)
- [ ] **Definiciones editables desde admin panel**: CRUD de exclusion lists, keywords, engines

## v0.4.0

- [ ] **Sistema de plugins**: scripts personalizados que se ejecutan contra ADB
- [ ] **Perfiles de dispositivo**: guardar y restaurar configuraciones por dispositivo
- [ ] **Programación de tareas**: bloquear/desbloquear notificaciones en horarios específicos
- [ ] **Exportación de reportes**: PDF/CSV con estado de procesos y apps

## v1.0.0

- [ ] **Interfaz nativa**: migrar a Tauri o Wails para distribución desde app stores
- [ ] **Multi-dispositivo simultáneo**: gestionar varios teléfonos a la vez
- [ ] **API pública**: REST API documentada para integraciones de terceros
- [ ] **Internacionalización completa**: +5 idiomas
