#pragma once

class IWasm
{
    virtual void init(char *kvDbPath) = 0;
    virtual void wasmRunVm(
        char *astPath,
        char *input,
        char *machineId) = 0;
    virtual void wasmRunEffects(char *effectsStr) = 0;
    virtual void wasmRunTrxs(
        char *astStorePath,
        char *input) = 0;
};
