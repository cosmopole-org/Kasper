import json

print("hey kasper !")

model = [
    [1, 2, 3],
    [4, 5, 6],
    [7, 8, 9],
]

with open("/app/output/model.data", 'wb') as outfile:
    json.dump(model, outfile)
