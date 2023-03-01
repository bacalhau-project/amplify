# 🔊 Amplify

Amplify attaches afterburners to your data. Amplify:

* **describes**: metadata detection and forwarding
* **augments**: derivative data generation like thumbnails, previews, etc. 
* **enriches**: batteries-included value-adds services like OCR, translation, transcription, etc.

Amplify leverages the decentralized compute provided by [Bacalhau](https://bacalhau.org) to magically enrich your data. A built-in suite of pipelines decides what your data is and how to best improve upon it. You can also self-host Amplify to trigger off your offline data sources and implement your own custom pipelines.

## Roadmap -- May 2023

* [ ] Amplify as a service -- a hosted version of Amplify that exposes REST APIs and a web UI for submitting, monitoring and inspecting Amplify jobs
* [ ] Amplify Job SDK -- Allow collaborators to enhance Amplify with new jobs
* [ ] Amplify Pipeline SDK -- High-level abstraction allowing jobs to be chained depending on mime type, results, etc.
* [ ] Amplify Trigger SDK -- Automated triggering and queuing of Amplify jobs based on new data announcements
    * [ ] Filecoin deal trigger
    * [ ] IPFS DHT trigger
    * [ ] IPFS PubSub trigger
    * [ ] HTTP watch trigger
    * [ ] IPFS stream trigger
* [ ] Amplify Contributor Repository -- Contributor-style repository for community-developed jobs and pipelines (inspired by [OpenTelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)