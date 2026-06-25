#ifndef CUSTOM_PDO_NAME_H
#define CUSTOM_PDO_NAME_H

//-------------------------------------------------------------------//
//                                                                   //
//     This file has been created by the Easy Configurator tool      //
//                                                                   //
//     Easy Configurator project easycat_safe_64.prj
//                                                                   //
//-------------------------------------------------------------------//


#define CUST_BYTE_NUM_OUT	64
#define CUST_BYTE_NUM_IN	64
#define TOT_BYTE_NUM_ROUND_OUT	64
#define TOT_BYTE_NUM_ROUND_IN	64


typedef union												//---- output buffer ----
{
	uint8_t  Byte [TOT_BYTE_NUM_ROUND_OUT];
	struct
	{
		uint32_t    out_word_00;
		uint32_t    out_word_01;
		uint32_t    out_word_02;
		uint32_t    out_word_03;
		uint32_t    out_word_04;
		uint32_t    out_word_05;
		uint32_t    out_word_06;
		uint32_t    out_word_07;
		uint32_t    out_word_08;
		uint32_t    out_word_09;
		uint32_t    out_word_10;
		uint32_t    out_word_11;
		uint32_t    out_word_12;
		uint32_t    out_word_13;
		uint32_t    out_word_14;
		uint32_t    out_word_15;
	}Cust;
} PROCBUFFER_OUT;


typedef union												//---- input buffer ----
{
	uint8_t  Byte [TOT_BYTE_NUM_ROUND_IN];
	struct
	{
		uint32_t    in_word_00;
		uint32_t    in_word_01;
		uint32_t    in_word_02;
		uint32_t    in_word_03;
		uint32_t    in_word_04;
		uint32_t    in_word_05;
		uint32_t    in_word_06;
		uint32_t    in_word_07;
		uint32_t    in_word_08;
		uint32_t    in_word_09;
		uint32_t    in_word_10;
		uint32_t    in_word_11;
		uint32_t    in_word_12;
		uint32_t    in_word_13;
		uint32_t    in_word_14;
		uint32_t    in_word_15;
	}Cust;
} PROCBUFFER_IN;

#endif