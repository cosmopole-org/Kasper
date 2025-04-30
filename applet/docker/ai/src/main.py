import json

print("hey kasper !")

model = [1, 2, 3]

with open("/app/output", 'w', encoding="utf8") as outfile:
    json.dump(model, outfile)
