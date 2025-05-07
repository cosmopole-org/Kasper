import tensorflow.keras as keras
import os

os.rename('/app/input/model1', '/app/input/model1.h5')
os.rename('/app/input/model2', '/app/input/model2.h5')

print("merge models")

model1 = keras.models.load_model('/app/input/model1.h5')
model2 = keras.models.load_model('/app/input/model2.h5')

def convert_to_layers(model_arch):
    rand_layers = []
    for layer in model_arch.layers:
        weights = layer.get_weights()
        rand_layer = []
        for sublayer in weights:
            rand_layer.append(sublayer)
        rand_layers.append(rand_layer)
    return rand_layers


model1_layers = convert_to_layers(model1)
model2_layers = convert_to_layers(model2)

layers = []
counter = 0
for i in range(0, len(model1_layers)):
    weights = model1_layers[i]
    sublayers = []
    for j in range(0, len(weights)):
        if counter % 2 == 0:
            sublayers.append(
                model1_layers[i][j]
            )
        else:
            sublayers.append(
                model2_layers[i][j]
            )
    layers.append(sublayers)

for i in range(0, len(layers)):
    model1.layers[i].set_weights(layers[i])

model1.save("/app/output.h5")
os.rename("/app/output.h5", "/app/output")