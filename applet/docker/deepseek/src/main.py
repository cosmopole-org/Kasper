import json

print("\n\n\n\n\n\n\n\n\n\nhey keyhan !\n\n\n\n\n\n\n\n\n\n")

with open("/app/input/model") as f:
    model = json.load(f)
    model = model * 5
    with open("/app/output", 'w', encoding="utf8") as outfile:
        json.dump(model, outfile)
