import json

print("merge models")

model1 = []
model2 = []
with open("/app/input/model1") as f:
    model1 = json.load(f)

with open("/app/input/model2") as f:
    model2 = json.load(f)

model = model1 + model2

with open("/app/output", 'w', encoding="utf8") as outfile:
    json.dump(model, outfile)
