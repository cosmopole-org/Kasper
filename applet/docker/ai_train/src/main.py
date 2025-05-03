import json

print("train model")

model = []
with open("/app/input/model") as f:
    model = json.load(f)

for i in range(0, len(model)):
    model[i] = model[i] + 10

with open("/app/output", 'w', encoding="utf8") as outfile:
    json.dump(model, outfile)
