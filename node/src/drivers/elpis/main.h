
#ifdef __cplusplus
extern "C"
{
#endif
    char *elpisCallback(char *);
    void runVm(
        const char *astPath,
        const char *sendType,
        const char *spaceId,
        const char *topicId,
        const char *memberId,
        const char *recvId,
        const char *inputData
    );
#ifdef __cplusplus
} // extern "C"
#endif
