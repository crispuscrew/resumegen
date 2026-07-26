#let r-lang = "en"
#let r-name = "Ivan Petrov"
#let r-summary = [Go engineer with a background in systems and C++ embedded\/robotics. Built production gRPC\/protobuf services, REST backends, and TUI tooling]
#let r-contacts = (
	(value : "ivan.petrov@example.com", href : "mailto:ivan.petrov@example.com"), 
	(value : "linkedin.com/in/yourprofile", href : "https://linkedin.com/in/yourprofile"), 
	(value : "github.com/yourhandle", href : "https://github.com/yourhandle"), 
)
#let r-jobs = (
	(title : "Software Engineer", date : "Jan. 2025 - Present", company : "Northwind Robotics", location : "Berlin, Germany", bullets : (
		[Built a Go service that generates *protobuf* schemas from C++ headers at build time and bridges *gRPC* \<-\> a legacy binary telemetry protocol: sustains *12k RPC\/s* on a single ARM board across a fleet of vehicles],
		[Refactored and extended a 2M+ LOC *Qt\/QML* ground control station for robotic systems: custom QML screens and C++ integrations in a 7-person team],
		[Designed and implemented an internal *artifact storage service* in Go: REST API, Redis for metadata, Docker volume as file store, consumed by a React frontend - replaced manual binary distribution],
		[Developed a USB-over-IP relay on a single-board *ARM Linux* computer for the radio subsystem, achieving *80 MB\/s* sustained throughput],)),
	(title : "Software Engineer (Contract)", date : "Sep. 2024 - Jan. 2025", company : "Contoso Avionics", location : "Remote", bullets : (
		[Designed system architecture and developed the *Flutter* operator interface with a custom *WebSocket* protocol for real-time ROS integration],)),
	(title : "CAE / Software Engineer", date : "Mar. 2023 - Aug. 2024", company : "Fabrikam Propulsion", location : "Munich, Germany", bullets : (
		[Developed *C++* firmware for an *STM32F407*-based engine test stand: sensor data acquisition, SD logging, and UART telemetry output],)),
)
#let r-projects = (
	(title : "nightglass", date : "", subtitle : "Go, GStreamer, ARM Linux, NPU", detail : "", bullets : (
		[Building a DIY digital night-vision helmet on a *6 TOPS NPU* board: dual thermal cameras fused with a near-infrared stream, on-device *YOLOv8n* person detection - targeting *`<100 ms`* end-to-end latency, primary language Go],)),
	(title : "pgscope", date : "", subtitle : "Go, TUI, PostgreSQL", detail : "github.com/yourhandle/pgscope", bullets : (
		[Keyboard-driven read-only TUI for exploring PostgreSQL, designed to run on the server and accessed over SSH - no port forwarding or local client setup],
		[Component-based *Bubble Tea* architecture: schema browser and live table data viewer],
		[All queries run in *REPEATABLE READ* read-only transactions - consistent snapshot with no risk of accidental mutation],)),
	(title : "vpn-ansible", date : "", subtitle : "Ansible, DevOps", detail : "github.com/yourhandle/vpn-ansible", bullets : (
		[Ansible playbook for automated *WireGuard* VPN provisioning in Docker: idempotent per-client key and config generation, stable IPs across re-deploys],
		[Multi-distro support \(Debian, RedHat, Arch, Alpine\) - from bare VPS to running VPN in a single command],)),
)
#let r-skills = (
	(category : "Languages", items : ([Go],[C\/C++],[Python],[SQL],)),
	(category : "Technologies", items : ([Qt\/QML],[WebSocket],[Ansible],)),
	(category : "Databases", items : ([PostgreSQL],[Redis],[SQLite],)),
	(category : "Developer Tools", items : ([Git],[GitHub Actions],[GitLab CI],[Docker \/ Podman],[Linux],)),
)
#let r-edu = (
	(title : "Example State University", degree : "M.Sc., Computational Mathematics and Computer Science", location : "Berlin, Germany", date : "Aug. 2020 - Jun. 2022"),
	(title : "Institute of Applied Technology", degree : "B.Sc., Geoinformatics, Expected 2029", location : "Munich, Germany", date : "Aug. 2022 - Jan. 2029"),
)
