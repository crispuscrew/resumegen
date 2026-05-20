#let r-lang = "en"
#let r-name = "Ruslan Gabzetdinov"
#let r-summary = [Go engineer with a background in systems and C++ embedded/robotics. Built production gRPC/protobuf services, REST backends, and TUI tooling]
#let r-contacts = (
	(value : "gabzetdinov@proton.me", href : "mailto:gabzetdinov@proton.me"), 
	(value : "linkedin.com/in/gabzetdinovri", href : "https://linkedin.com/in/gabzetdinovri"), 
	(value : "github.com/crispuscrew", href : "https://github.com/crispuscrew"), 
)
#let r-jobs = (
	(title : "Software Engineer", date : "Jan. 2025 – Present", company : "LLC Gumich RTK", location : "Moscow, Russia", bullets : (
		[Built a Go microservice that auto-generates *protobuf* schemas from MAVLink C++ headers at build time and bridges *gRPC* ↔ MAVLink v2 (proprietary): sustains *12k RPC/s* peak on Orange Pi as part of a swarm/mesh multi-vehicle platform],
		[Refactored and extended a 2M+ LOC *Qt/QML* ground control station for robotic systems: custom QML screens and C++ integrations in a 7-person team],
		[Designed and implemented an internal *artifact storage service* in Go: REST API, Redis for metadata, Docker volume as file store, consumed by a React frontend - replaced manual binary distribution],
		[Developed a USB-over-IP relay on *Orange Pi* (ARM Linux) for the SDR subsystem, achieving *80 MB/s* sustained I/Q throughput],)),
	(title : "Software Engineer (Contract)", date : "Sep. 2024 – Jan. 2025", company : "LLC Ground Avionics", location : "Moscow, Russia", bullets : (
		[Designed system architecture and developed the *Flutter* operator interface with a custom *WebSocket* protocol for real-time ROS integration],)),
	(title : "CAE / Software Engineer", date : "Mar. 2023 – Aug. 2024", company : "LLC Indicative Engines", location : "Sarov / Moscow, Russia", bullets : (
		[Developed *C++* firmware for an *STM32F407*-based liquid rocket engine test stand: sensor data acquisition, SD logging, and UART telemetry output],)),
)
#let r-projects = (
	(title : "HAVEN", date : "", subtitle : "Go, GStreamer, ARM Linux, RKNN", detail : "", bullets : (
		[Building a DIY digital NVG helmet on *RK3588* (Orange Pi 5B): dual CVBS thermal cameras fused with a NIR stream, *RKNN/YOLOv8n* person detection on the 6 TOPS NPU - targeting *`<100 ms`* end-to-end latency, primary language Go],)),
	(title : "pgxray", date : "", subtitle : "Go, TUI, PostgreSQL", detail : "github.com/crispuscrew/pgxray", bullets : (
		[Keyboard-driven read-only TUI for exploring PostgreSQL, designed to run on the server and accessed over SSH - no port forwarding or local client setup],
		[Component-based *Bubble Tea v2* architecture: schema browser and live table data viewer],
		[All queries run in *REPEATABLE READ* read-only transactions - consistent snapshot with no risk of accidental mutation],)),
	(title : "amnezia-ansible", date : "", subtitle : "Ansible, DevOps", detail : "github.com/crispuscrew/amnezia-ansible", bullets : (
		[Ansible playbook for automated *AmneziaWireGuard* VPN provisioning in Docker: idempotent per-client key and config generation, stable IPs across re-deploys],
		[Multi-distro support (Debian, RedHat, Arch, Alpine) - from bare VPS to running VPN in a single command],)),
)
#let r-skills = (
	(category : "Languages", items : ([Go],[C/C++],[Python],[SQL],)),
	(category : "Technologies", items : ([Qt/QML],[WebSocket],[Ansible],)),
	(category : "Databases", items : ([PostgreSQL],[Redis],[SQLite],)),
	(category : "Developer Tools", items : ([Git],[GitHub Actions],[GitLab CI],[Docker / Podman],[Linux],)),
)
#let r-edu = (
	(title : "M. V. Lomonosov Moscow State University", degree : "Diploma, Computational Mathematics and Computer Science", location : "Moscow, Russia", date : "Aug. 2020 – Jun. 2022"),
	(title : "National University of Science and Technology MISIS", degree : "Specialist in Geoinformatics, Expected 2029", location : "Moscow, Russia", date : "Aug. 2022 – Jan. 2029"),
)
