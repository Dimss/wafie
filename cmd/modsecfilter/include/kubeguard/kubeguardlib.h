#ifndef KUBEGUARDLIB_LIBRARY_H
#define KUBEGUARDLIB_LIBRARY_H
#include <modsecurity/transaction.h>

typedef struct {
    const unsigned char *key;
    const unsigned char *value;
} EvaluationRequestHeader;

typedef struct {
    char *client_ip;
    char *uri;
    char *http_method;
    char *http_version;
    char *body;
    size_t headers_count;
    EvaluationRequestHeader *headers;
    Transaction *transaction;
} EvaluationRequest;

void kg_library_init(char const *config_path);

int kg_process_request_headers(EvaluationRequest const *request);

int kg_process_request_body(EvaluationRequest const *request);

void kg_init_request_transaction(EvaluationRequest *request);

void kg_transaction_cleanup(EvaluationRequest const *request);

void kg_dump_rules();

void kg_cleanup(char const *error, RulesSet *rules, ModSecurity *modsec);

int kg_add_rule(char const *rule);

#endif //KUBEGUARDLIB_LIBRARY_H
