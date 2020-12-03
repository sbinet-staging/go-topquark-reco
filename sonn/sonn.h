#ifndef SONN_H
#define SONN_H 1

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
	double Px, Py, Pz, E;
} tlv_t;

typedef struct {
	int n;
	tlv_t *tlvs;
} tlvs_t;

tlv_t get_tlv(tlv_t *tlvs, int i);

tlvs_t sonn(
		tlv_t lep, tlv_t lepbar, int pdgID_lep, int pdgID_lepbar,
		tlv_t jet, tlv_t jetbar, int isb_jet, int isb_jetbar,
		double emissx, double emissy);

void load_smearing_histos(const char* fname);

void init_logs();
void stop_logs();

#ifdef __cplusplus
} // extern "C"
#endif

#endif // SONN_H
