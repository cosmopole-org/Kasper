import keras
import json
import pandas as pd
from tensorflow.keras.utils import to_categorical
from sklearn.model_selection import KFold
import os
import tensorflow as tf

os.environ['CUDA_VISIBLE_DEVICES'] = '-1'
tf.compat.v1.enable_eager_execution()

print("train model")

csv_file_path = "/app/input/train.csv"
test_csv_file_path = "/app/input/test.csv"

data = pd.read_csv(csv_file_path)
test_data = pd.read_csv(test_csv_file_path)

activity_mapping = {
    'WALKING': 1,
    'WALKING_UPSTAIRS': 2,
    'WALKING_DOWNSTAIRS': 3,
    'SITTING': 4,
    'STANDING': 5,
    'LAYING': 6
}

data['Activity_numeric'] = data['Activity'].map(activity_mapping)
test_data['Activity_numeric'] = test_data['Activity'].map(activity_mapping)

data_central = data[data['subject'] < 26]
databases = {}

for subject_number in range(26, 31):
    databases[subject_number] = data[data['subject'] == subject_number]


def preprocess_data(dataset, num_classes):
    X = dataset.drop(columns=['Activity', 'Activity_numeric']).to_numpy()
    X_train = X.reshape(X.shape[0], X.shape[1], 1)
    y = dataset['Activity_numeric'].to_numpy()
    y_one_hot = to_categorical(y - 1, num_classes=num_classes)
    return X_train, y_one_hot


X, y_one_hot = preprocess_data(data_central, num_classes=6)
test_X, test_y_one_hot = preprocess_data(test_data, num_classes=6)

preprocessed_federated_data = {}

for dataset_index in range(26, 31):
    temp_X, temp_y_one_hot = preprocess_data(
        databases[dataset_index], num_classes=6)
    kf = KFold(n_splits=5, shuffle=True, random_state=42)
    fold_index = 0
    for train_index, valid_index in kf.split(temp_X):
        X_train, X_valid = temp_X[train_index], temp_X[valid_index]
        y_train, y_valid = temp_y_one_hot[train_index], temp_y_one_hot[valid_index]
        preprocessed_federated_data[f'{dataset_index}_fold_{fold_index}'] = {
            'X_train': X_train,
            'X_valid': X_valid,
            'y_train': y_train,
            'y_valid': y_valid
        }
        fold_index += 1

FOLD_INDEX = 3

idVal = {}
with open("/app/input/id") as f:
    idVal = json.load(f)

FIRST_CLIENT = 26
client_id = FIRST_CLIENT + idVal['value']

x_local, y_local = preprocessed_federated_data[f'{client_id}_fold_{FOLD_INDEX}'][
    'X_train'], preprocessed_federated_data[f'{client_id}_fold_{FOLD_INDEX}']['y_train']
x_local_valid, y_local_valid = preprocessed_federated_data[f'{client_id}_fold_{FOLD_INDEX}'][
    'X_valid'], preprocessed_federated_data[f'{client_id}_fold_{FOLD_INDEX}']['y_valid']

os.rename('/app/input/model', '/app/input/model.h5')

model_arch = keras.models.load_model('/app/input/model.h5')

dataset = [x_local, y_local, x_local_valid, y_local_valid]

with tf.device("CPU:0"):
    model_arch.fit(
        dataset[0],
        dataset[1],
        validation_data=(dataset[2], dataset[3]),
        epochs=40,
        batch_size=64,
        validation_split=0.3,
    )

model_arch.save("/app/output.h5")
os.rename("/app/output.h5", "/app/output")
