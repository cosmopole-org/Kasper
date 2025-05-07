import tensorflow as tf
import pandas as pd
from keras.models import Sequential
from keras.layers import Conv1D, MaxPooling1D, Flatten, Dense
from keras.utils import to_categorical
from sklearn.model_selection import KFold
import os

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

print("init model")

FOLD_INDEX = 3

feature_shape = X.shape[1]
classes = 6
lr = 0.01

def build_model(feature_shape, classes=6):
    model = Sequential()
    model.add(Conv1D(filters=32, kernel_size=3,
              activation='relu',
              input_shape=(feature_shape)))
    model.add(MaxPooling1D(pool_size=2))
    model.add(Conv1D(filters=64, kernel_size=3,
                     activation='relu'))
    model.add(MaxPooling1D(pool_size=2))
    model.add(Flatten())
    model.add(Dense(128, activation='relu'))
    model.add(Dense(64, activation='relu'))
    model.add(Dense(classes, activation='softmax'))
    model.compile(optimizer=tf.keras.optimizers.SGD(lr),
                  loss='categorical_crossentropy',
                  metrics=['accuracy'])
    return model

model_arch = build_model(X.shape[1:], classes)
model_arch.save("/app/output.h5")
os.rename("/app/output.h5", "/app/output")