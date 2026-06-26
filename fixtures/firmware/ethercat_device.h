#ifndef LAN9252_BASIC_ETHERCAT_DEVICE_H
#define LAN9252_BASIC_ETHERCAT_DEVICE_H

#include <stdint.h>

#define CUST_BYTE_NUM_OUT	3
#define CUST_BYTE_NUM_IN	3
#define TOT_BYTE_NUM_ROUND_OUT	3
#define TOT_BYTE_NUM_ROUND_IN	3


typedef union
{
	uint8_t Byte[TOT_BYTE_NUM_ROUND_OUT];
	struct
	{
		uint16_t control_word;
		uint8_t reserved_tail[1];
	} Cust;
} PROCBUFFER_OUT;


typedef union
{
	uint8_t Byte[TOT_BYTE_NUM_ROUND_IN];
	struct
	{
		uint16_t status_word;
		uint8_t reserved_tail[1];
	} Cust;
} PROCBUFFER_IN;


#endif /* LAN9252_BASIC_ETHERCAT_DEVICE_H */
