import tensorflow.keras as keras
import numpy as np
import os

os.rename('/app/input/model1', '/app/input/model1.h5')
os.rename('/app/input/model2', '/app/input/model2.h5')

print("aggregate models")

model1 = keras.models.load_model('/app/input/model1.h5')
model2 = keras.models.load_model('/app/input/model2.h5')

local_models = [model1, model2]

layers = []
for i in range(0, len(local_models[0])):
    sublayers = []
    for j in range(0, len(local_models[0][i])):
        sublayers.append(np.mean(np.array([model[i][j] for model in local_models]), axis=0))
    layers.append(sublayers)

for i in range(0, len(layers)):
    model1.layers[i].set_weights(layers[i])

model1.save("/app/output.h5")
os.rename("/app/output.h5", "/app/output")