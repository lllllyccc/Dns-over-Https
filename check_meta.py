import json
d = json.load(open("/tmp/meta.json"))
print("vnicId:", d.get("vnicId","?"))
print("compartmentId:", d.get("compartmentId","?"))
print("subnetId:", d.get("subnetId","?"))
