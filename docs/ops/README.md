# Operational Docs

## Disabling all triggered jobs

Sometimes we need to stop all jobs to aid debugging bacalhau. The simplest way to do this is by disabling all triggers. To do this, add an override to the systemd service.

First connect to the VM of interest:

```bash
gcloud compute ssh amplify-vm-production-0 -- bash
```

Then create an override file and restart the service:

```bash
sudo mkdir -p /etc/systemd/system/amplify.service.d/
sudo tee /etc/systemd/system/amplify.service.d/disable-triggers.conf > /dev/null <<'EOI'
[Service]
Environment="AMPLIFY_TRIGGER_IPFS_SEARCH_ENABLED=false"
EOI
sudo systemctl daemon-reload
sudo systemctl restart amplify
```
You can verify that it has successfully restarted by checking the logs:

```bash
journalctl -u amplify -f
```

And [viewing the API](http://amplify.bacalhau.org/api/v0/queue).

### Re-enabling Triggered jobs

Just delete the override file and restart.

```bash
gcloud compute ssh amplify-vm-production-0 -- bash
```

```bash
sudo rm -f /etc/systemd/system/amplify.service.d/disable-triggers.conf
sudo systemctl daemon-reload
sudo systemctl restart amplify
```

