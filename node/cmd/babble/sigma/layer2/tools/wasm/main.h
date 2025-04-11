
#ifdef __cplusplus
extern "C"
{
#endif
    char *wasmCallback(char *);
    void init();
    void wasmRunVm(
        char *astPath,
        char *input,
        char *machineId);
    void wasmRunEffects(char *effectsStr);
    void wasmRunTrxs(
        char *astStorePath,
        char *input);
#ifdef __cplusplus
} // extern "C"
#endif
