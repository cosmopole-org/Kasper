import json

print("init model")

model = [1, 2, 3, 4, 5, 6]

with open("/app/output", 'w', encoding="utf8") as outfile:
    json.dump(model, outfile)
