#include <resolv.h>

int __res_nquery(res_state statp, const char *dname, int class, int type, u_char *answer, int anslen) {
    return res_nquery(statp, dname, class, type, answer, anslen);
}

int __dn_expand(const u_char *msg, const u_char *eomorig, const u_char *comp_dn, char *exp_dn, int length) {
    return dn_expand(msg, eomorig, comp_dn, exp_dn, length);
}
