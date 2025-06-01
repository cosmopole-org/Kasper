#pragma once

#include <stdio.h>
#include <unordered_map>
#include <string>
#include <list>
#include <unistd.h>
#include <mutex>
#include <string.h>
#include <any>
#include <unordered_set>
#include <condition_variable>
#include <queue>
#include <thread>
#include <functional>
#include <iostream>
#include <fstream>

#include "iwasm.h"
#include "lib/runtime.h"

using namespace std;

class Wasm : public IWasm
{
  void init(char *kvDbPath) override;
  void wasmRunVm(
      char *astPath,
      char *input,
      char *machineId) override;
  void wasmRunEffects(char *effectsStr) override;
  void wasmRunTrxs(
      char *astStorePath,
      char *input) override;
};
