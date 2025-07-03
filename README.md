# `unifi-discovery`

Grafana Alloy [discovery.http]-compatible server for the Unifi Network API.

Runs on the Raspberry Pi 5 in our office and exposes IP addresses of our Unifi devices to Grafana Alloy on the same server.

[discovery.http]: https://grafana.com/docs/alloy/latest/reference/components/discovery/discovery.http/

Example output:
```jsonc
[
  {
    "targets": [
      "85.252.137.188"
    ],
    "labels": {
      "device_id": "98c4be62-7bd9-31a6-8e22-86df5c3fc625",
      "device_model": "UDM Pro",
      "device_name": "R9-5A51-UDM-9G-2S+RM"
    }
  },
  {
    "targets": [
      "192.168.0.106"
    ],
    "labels": {
      "device_id": "98ec0aeb-efa0-32e3-a055-c1550439d72e",
      "device_model": "USW Pro 48",
      "device_name": "R9-5A51-USW-48G-4S+RM"
    }
  },
  # ...
]
```