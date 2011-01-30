package threefish

import (
	"os"
	"strconv"
	"encoding/binary"
)

const KEY_SCHEDULE_CONST = uint64(0x1BD11BDAA9FC1A22)
const EXPANDED_TWEAK_SIZE = 3;


// Internal interface to simplify Threefish usage
type cipherInternal interface {
    /**
     * Encrypt function
     * 
     * Derived classes must implement this function.
     * 
     * @param input
     *     The plaintext input.
     * @param output
     *     The ciphertext output.
     */
    encrypt(input, output [] uint64)

    /**
     * Decrypt function
     * 
     * Derived classes must implement this function.
     * 
     * @param input
     *     The ciphertext input.
     * @param output
     *     The plaintext output.
     */
    decrypt(input, output []uint64)
    
    getTempData() ([]uint64, []uint64)
    setTweak(tweak []uint64)
    setKey(key []uint64)
}


// A Cipher is an instance of Threefish using a particular key and state size.
type Cipher struct {
	stateSize int
	internal cipherInternal
}

type KeySizeError int

func (k KeySizeError) String() string {
	return "crypto/threefish: invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a Cipher.
// The key argument should be the Threefish key, 32, 64 or 128 bytes.
func NewCipher(key []byte, tweak []uint64) (*Cipher, os.Error) {
    c := new(Cipher)

    var err os.Error

    switch len(key) {
    case 32:
        c.internal, err = newThreefish256(key, tweak)
    case 64:
        c.internal, err = newThreefish512(key, tweak)
    case 128:
        c.internal, err = newThreefish1024(key, tweak)
    default:
        return nil, KeySizeError(len(key))
    }		
    c.stateSize = len(key) * 8
    return c, err
}

// NewCipher creates and returns a Cipher.
// The key argument should be the Threefish key, 4, 8 or 16 uint64
func NewCipher64(key, tweak []uint64) (*Cipher, os.Error) {
    c := new(Cipher)

    var err os.Error
    switch len(key) {
    case 4:
        c.internal, err = newThreefish256_64(key, tweak)
    case 8:
        c.internal, err = newThreefish512_64(key, tweak)
    case 16:
        c.internal, err = newThreefish1024_64(key, tweak)
    default:
        return nil, KeySizeError(len(key))
    }       
    c.stateSize = len(key) * 8
    return c, err
}

// NewCipher creates and returns a Cipher.
// The key argument should be the Threefish key, 32, 64 or 128 bytes.
func NewCipherSize(size int) (*Cipher, os.Error) {
    c := new(Cipher)
        
    var err os.Error
        
    switch size {
    case 256:
        c.internal, err = newThreefish256(nil, nil)
    case 512:
        c.internal, err = newThreefish512(nil, nil)
    case 1024:
        c.internal, err = newThreefish1024(nil, nil)
    default:
        return nil, KeySizeError(size)
    }       
    c.stateSize = size
    return c, err
}

// BlockSize returns the cipher's block size in bytes.
func (c *Cipher) BlockSize() int {
    return c.stateSize / 8
}


// Encrypt encrypts the first block in src into dst.
// Dst and src may point at the same memory.
func (c *Cipher) Encrypt(dst, src []byte) {

    uintLen := c.stateSize / 64

    // This saves makes
    tmpin, tmpout := c.internal.getTempData()
    
    for i := 0; i < uintLen; i++ {
        tmpin[i] = binary.LittleEndian.Uint64(src[i*8:i*8+8])
    }
    c.internal.encrypt(tmpin, tmpout)

    for i := 0; i < uintLen; i++ {
        binary.LittleEndian.PutUint64(dst[i*8:i*8+8], tmpout[i])
    }
}

// Decrypt decrypts the first block in src into dst.
// Dst and src may point at the same memory.
func (c *Cipher) Decrypt(dst, src []byte) {

    uintLen := c.stateSize / 64
    
    // This saves a make
    tmpin, tmpout := c.internal.getTempData()
    
    for i := 0; i < uintLen; i++ {
        tmpin[i] = binary.LittleEndian.Uint64(src[i*8:i*8+8])
    }
    c.internal.decrypt(tmpin, tmpout)

    for i := 0; i < uintLen; i++ {
        binary.LittleEndian.PutUint64(dst[i*8:i*8+8], tmpout[i])
    }
}

// Encrypt encrypts the first block in src into dst, blocks
// are unit64 arrays
// Dst and src may point at the same memory.
func (c *Cipher) Encrypt64(dst, src []uint64) {
    c.internal.encrypt(src, dst)
}

// Decrypt encrypts the first block in src into dst, blocks
// are unit64 arrays
// Dst and src may point at the same memory.
func (c *Cipher) Decrypt64(dst, src []uint64) {
    c.internal.decrypt(src, dst)
}

func (c *Cipher) SetTweak(tweak []uint64) {
    c.internal.setTweak(tweak)
}

func (c *Cipher) SetKey(key []uint64) {
    c.internal.setKey(key)
}

// Some helper functions
/**
 * Initialize the tweak data
 */
func setTweak(tweak, expTweak []uint64) {
    if tweak != nil {
        expTweak[0] = tweak[0];
        expTweak[1] = tweak[1];
        expTweak[2] = tweak[0] ^ tweak[1];
    }
}

/**
 * Expand the key data
 */
func setKey(key, expKey []uint64) {
    var i int
    parity := uint64(KEY_SCHEDULE_CONST)

    for i = 0; i < len(expKey) - 1; i++ {
        expKey[i] = key[i];
        parity ^= key[i];
    }
    expKey[i] = parity;
}

/*
 * Threefish 256 implementation
 */
const CIPHER_SIZE_256 = 256
const CIPHER_QWORDS_256 = CIPHER_SIZE_256 / 64
const EXPANDED_KEY_SIZE_256 = CIPHER_QWORDS_256 + 1

type threefish256 struct {
	expanedTweak [EXPANDED_TWEAK_SIZE]uint64
	expanedKey [EXPANDED_KEY_SIZE_256]uint64
	tmpData1, tmpData2  [CIPHER_QWORDS_256]uint64
}

// Get an initialized Threefish256 structure
func newThreefish256(key []byte, tweak []uint64) (*threefish256, os.Error) {
    c := new(threefish256)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_256 - 1] = KEY_SCHEDULE_CONST

    // Convert key bytes to required uint64 array
    if key != nil {
        var tmpkey [EXPANDED_KEY_SIZE_256]uint64		
        for i := 0; i < EXPANDED_KEY_SIZE_256 - 1; i++ {
            tmpkey[i] = binary.LittleEndian.Uint64(key[i*8:i*8+8])
        }
        setKey(tmpkey[:], c.expanedKey[:])
    }
    return c, nil
}

// Get an initialized Threefish256 structure
func newThreefish256_64(key, tweak []uint64) (*threefish256, os.Error) {
    c := new(threefish256)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_256 - 1] = KEY_SCHEDULE_CONST

    setKey(key, c.expanedKey[:])
    return c, nil
}

func (tf *threefish256) getTempData() ([]uint64, []uint64) {
    return tf.tmpData1[:], tf.tmpData2[:]
}

func (tf *threefish256) setTweak(tweak []uint64) {
    setTweak(tweak, tf.expanedTweak[:])    
}

func (tf *threefish256) setKey(key []uint64) {
    setKey(key, tf.expanedKey[:])    
}

func (tf *threefish256) encrypt(input, output [] uint64) {
    
   b0 := input[0]
   b1 := input[1]
   b2 := input[2]
   b3 := input[3]
   
   k0 := tf.expanedKey[0]
   k1 := tf.expanedKey[1]
   k2 := tf.expanedKey[2] 
   k3 := tf.expanedKey[3]
   k4 := tf.expanedKey[4]
   
   t0 := tf.expanedTweak[0]
   t1 := tf.expanedTweak[1]
   t2 := tf.expanedTweak[2]
   
   b1 += k1 + t0; b0 += b1 + k0; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k3; b2 += b3 + k2 + t1; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k2 + t1; b0 += b1 + k1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k4 + 1; b2 += b3 + k3 + t2; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k3 + t2; b0 += b1 + k2; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k0 + 2; b2 += b3 + k4 + t0; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k4 + t0; b0 += b1 + k3; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k1 + 3; b2 += b3 + k0 + t1; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k0 + t1; b0 += b1 + k4; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k2 + 4; b2 += b3 + k1 + t2; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k1 + t2; b0 += b1 + k0; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k3 + 5; b2 += b3 + k2 + t0; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k2 + t0; b0 += b1 + k1; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k4 + 6; b2 += b3 + k3 + t1; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k3 + t1; b0 += b1 + k2; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k0 + 7; b2 += b3 + k4 + t2; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k4 + t2; b0 += b1 + k3; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k1 + 8; b2 += b3 + k0 + t0; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k0 + t0; b0 += b1 + k4; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k2 + 9; b2 += b3 + k1 + t1; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k1 + t1; b0 += b1 + k0; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k3 + 10; b2 += b3 + k2 + t2; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k2 + t2; b0 += b1 + k1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k4 + 11; b2 += b3 + k3 + t0; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k3 + t0; b0 += b1 + k2; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k0 + 12; b2 += b3 + k4 + t1; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k4 + t1; b0 += b1 + k3; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k1 + 13; b2 += b3 + k0 + t2; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k0 + t2; b0 += b1 + k4; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k2 + 14; b2 += b3 + k1 + t0; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k1 + t0; b0 += b1 + k0; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k3 + 15; b2 += b3 + k2 + t1; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;
   b1 += k2 + t1; b0 += b1 + k1; b1 = ((b1 << 14) | (b1 >> (64 - 14))) ^ b0;
   b3 += k4 + 16; b2 += b3 + k3 + t2; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b2;
   b0 += b3; b3 = ((b3 << 52) | (b3 >> (64 - 52))) ^ b0;
   b2 += b1; b1 = ((b1 << 57) | (b1 >> (64 - 57))) ^ b2;
   b0 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b0;
   b2 += b3; b3 = ((b3 << 40) | (b3 >> (64 - 40))) ^ b2;
   b0 += b3; b3 = ((b3 << 5) | (b3 >> (64 - 5))) ^ b0;
   b2 += b1; b1 = ((b1 << 37) | (b1 >> (64 - 37))) ^ b2;
   b1 += k3 + t2; b0 += b1 + k2; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b0;
   b3 += k0 + 17; b2 += b3 + k4 + t0; b3 = ((b3 << 33) | (b3 >> (64 - 33))) ^ b2;
   b0 += b3; b3 = ((b3 << 46) | (b3 >> (64 - 46))) ^ b0;
   b2 += b1; b1 = ((b1 << 12) | (b1 >> (64 - 12))) ^ b2;
   b0 += b1; b1 = ((b1 << 58) | (b1 >> (64 - 58))) ^ b0;
   b2 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b2;
   b0 += b3; b3 = ((b3 << 32) | (b3 >> (64 - 32))) ^ b0;
   b2 += b1; b1 = ((b1 << 32) | (b1 >> (64 - 32))) ^ b2;

   output[0] = b0 + k3;
   output[1] = b1 + k4 + t0;
   output[2] = b2 + k0 + t1;
   output[3] = b3 + k1 + 18;

}

func (tf *threefish256) decrypt(input, output []uint64) {
    
   b0 := input[0]
   b1 := input[1]
   b2 := input[2]
   b3 := input[3]
   
   k0 := tf.expanedKey[0]
   k1 := tf.expanedKey[1]
   k2 := tf.expanedKey[2] 
   k3 := tf.expanedKey[3]
   k4 := tf.expanedKey[4]
   
   t0 := tf.expanedTweak[0]
   t1 := tf.expanedTweak[1]
   t2 := tf.expanedTweak[2]

   var tmp uint64

   b0 -= k3;
   b1 -= k4 + t0;
   b2 -= k0 + t1;
   b3 -= k1 + 18;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k2; b1 -= k3 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k4 + t0; b3 -= k0 + 17;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k1; b1 -= k2 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k3 + t2; b3 -= k4 + 16;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k0; b1 -= k1 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k2 + t1; b3 -= k3 + 15;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k4; b1 -= k0 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k1 + t0; b3 -= k2 + 14;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k3; b1 -= k4 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k0 + t2; b3 -= k1 + 13;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k2; b1 -= k3 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k4 + t1; b3 -= k0 + 12;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k1; b1 -= k2 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k3 + t0; b3 -= k4 + 11;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k0; b1 -= k1 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k2 + t2; b3 -= k3 + 10;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k4; b1 -= k0 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k1 + t1; b3 -= k2 + 9;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k3; b1 -= k4 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k0 + t0; b3 -= k1 + 8;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k2; b1 -= k3 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k4 + t2; b3 -= k0 + 7;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k1; b1 -= k2 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k3 + t1; b3 -= k4 + 6;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k0; b1 -= k1 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k2 + t0; b3 -= k3 + 5;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k4; b1 -= k0 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k1 + t2; b3 -= k2 + 4;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k3; b1 -= k4 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k0 + t1; b3 -= k1 + 3;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k2; b1 -= k3 + t2;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k4 + t0; b3 -= k0 + 2;
   tmp = b3 ^ b0; b3 = (tmp >> 32) | (tmp << (64 - 32)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 32) | (tmp << (64 - 32)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 58) | (tmp << (64 - 58)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 22) | (tmp << (64 - 22)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 12) | (tmp << (64 - 12)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 25) | (tmp << (64 - 25)); b0 -= b1 + k1; b1 -= k2 + t1;
   tmp = b3 ^ b2; b3 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b3 + k3 + t2; b3 -= k4 + 1;
   tmp = b3 ^ b0; b3 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 37) | (tmp << (64 - 37)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 23) | (tmp << (64 - 23)); b0 -= b1;
   tmp = b3 ^ b2; b3 = (tmp >> 40) | (tmp << (64 - 40)); b2 -= b3;
   tmp = b3 ^ b0; b3 = (tmp >> 52) | (tmp << (64 - 52)); b0 -= b3;
   tmp = b1 ^ b2; b1 = (tmp >> 57) | (tmp << (64 - 57)); b2 -= b1;
   tmp = b1 ^ b0; b1 = (tmp >> 14) | (tmp << (64 - 14)); b0 -= b1 + k0; b1 -= k1 + t0;
   tmp = b3 ^ b2; b3 = (tmp >> 16) | (tmp << (64 - 16)); b2 -= b3 + k2 + t1; b3 -= k3;

   output[0] = b0;
   output[1] = b1;
   output[2] = b2;
   output[3] = b3;
}


/*
 * Threefish 512 implementation
 */
const CIPHER_SIZE_512 = 512
const CIPHER_QWORDS_512 = CIPHER_SIZE_512 / 64
const EXPANDED_KEY_SIZE_512 = CIPHER_QWORDS_512 + 1

type threefish512 struct {
    expanedTweak [EXPANDED_TWEAK_SIZE]uint64
    expanedKey [EXPANDED_KEY_SIZE_512]uint64
    tmpData1, tmpData2  [CIPHER_QWORDS_512]uint64
}

// Get an initialized Threefish512 structure
func newThreefish512(key []byte, tweak []uint64) (*threefish512, os.Error) {
    c := new(threefish512)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_512 - 1] = KEY_SCHEDULE_CONST

    // Convert key bytes to required uint64 array
    if key != nil {
        var tmpkey [EXPANDED_KEY_SIZE_512]uint64        
        for i := 0; i < EXPANDED_KEY_SIZE_512 - 1; i++ {
            tmpkey[i] = binary.LittleEndian.Uint64(key[i*8:i*8+8])
        }
        setKey(tmpkey[:], c.expanedKey[:]);
    }
    return c, nil
}

// Get an initialized Threefish512 structure
func newThreefish512_64(key, tweak []uint64) (*threefish512, os.Error) {
    c := new(threefish512)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_512 - 1] = KEY_SCHEDULE_CONST

    setKey(key, c.expanedKey[:]);
    return c, nil
}

func (tf *threefish512) getTempData() ([]uint64, []uint64) {
    return tf.tmpData1[:], tf.tmpData2[:]
}

func (tf *threefish512) setTweak(tweak []uint64) {
    setTweak(tweak, tf.expanedTweak[:])    
}

func (tf *threefish512) setKey(key []uint64) {
    setKey(key, tf.expanedKey[:])    
}

func (tf *threefish512) encrypt(input, output [] uint64) {
    
    b0 := input[0]; b1 := input[1]
    b2 := input[2]; b3 := input[3]
    b4 := input[4]; b5 := input[5]
    b6 := input[6]; b7 := input[7]

    k0 := tf.expanedKey[0]; k1 := tf.expanedKey[1]
    k2 := tf.expanedKey[2]; k3 := tf.expanedKey[3]
    k4 := tf.expanedKey[4]; k5 := tf.expanedKey[5]
    k6 := tf.expanedKey[6]; k7 := tf.expanedKey[7]
    k8 := tf.expanedKey[8];
      
    t0 := tf.expanedTweak[0]
    t1 := tf.expanedTweak[1]
    t2 := tf.expanedTweak[2]

    b1 += k1; b0 += b1 + k0; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k3; b2 += b3 + k2; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k5 + t0; b4 += b5 + k4; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k7; b6 += b7 + k6 + t1; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k2; b0 += b1 + k1; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k4; b2 += b3 + k3; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k6 + t1; b4 += b5 + k5; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k8 + 1; b6 += b7 + k7 + t2; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k3; b0 += b1 + k2; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k5; b2 += b3 + k4; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k7 + t2; b4 += b5 + k6; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k0 + 2; b6 += b7 + k8 + t0; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k4; b0 += b1 + k3; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k6; b2 += b3 + k5; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k8 + t0; b4 += b5 + k7; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k1 + 3; b6 += b7 + k0 + t1; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k5; b0 += b1 + k4; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k7; b2 += b3 + k6; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k0 + t1; b4 += b5 + k8; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k2 + 4; b6 += b7 + k1 + t2; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k6; b0 += b1 + k5; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k8; b2 += b3 + k7; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k1 + t2; b4 += b5 + k0; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k3 + 5; b6 += b7 + k2 + t0; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k7; b0 += b1 + k6; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k0; b2 += b3 + k8; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k2 + t0; b4 += b5 + k1; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k4 + 6; b6 += b7 + k3 + t1; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k8; b0 += b1 + k7; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k1; b2 += b3 + k0; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k3 + t1; b4 += b5 + k2; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k5 + 7; b6 += b7 + k4 + t2; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k0; b0 += b1 + k8; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k2; b2 += b3 + k1; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k4 + t2; b4 += b5 + k3; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k6 + 8; b6 += b7 + k5 + t0; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k1; b0 += b1 + k0; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k3; b2 += b3 + k2; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k5 + t0; b4 += b5 + k4; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k7 + 9; b6 += b7 + k6 + t1; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k2; b0 += b1 + k1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k4; b2 += b3 + k3; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k6 + t1; b4 += b5 + k5; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k8 + 10; b6 += b7 + k7 + t2; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k3; b0 += b1 + k2; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k5; b2 += b3 + k4; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k7 + t2; b4 += b5 + k6; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k0 + 11; b6 += b7 + k8 + t0; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k4; b0 += b1 + k3; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k6; b2 += b3 + k5; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k8 + t0; b4 += b5 + k7; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k1 + 12; b6 += b7 + k0 + t1; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k5; b0 += b1 + k4; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k7; b2 += b3 + k6; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k0 + t1; b4 += b5 + k8; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k2 + 13; b6 += b7 + k1 + t2; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k6; b0 += b1 + k5; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k8; b2 += b3 + k7; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k1 + t2; b4 += b5 + k0; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k3 + 14; b6 += b7 + k2 + t0; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k7; b0 += b1 + k6; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k0; b2 += b3 + k8; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k2 + t0; b4 += b5 + k1; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k4 + 15; b6 += b7 + k3 + t1; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;
    b1 += k8; b0 += b1 + k7; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b0;
    b3 += k1; b2 += b3 + k0; b3 = ((b3 << 36) | (b3 >> (64 - 36))) ^ b2;
    b5 += k3 + t1; b4 += b5 + k2; b5 = ((b5 << 19) | (b5 >> (64 - 19))) ^ b4;
    b7 += k5 + 16; b6 += b7 + k4 + t2; b7 = ((b7 << 37) | (b7 >> (64 - 37))) ^ b6;
    b2 += b1; b1 = ((b1 << 33) | (b1 >> (64 - 33))) ^ b2;
    b4 += b7; b7 = ((b7 << 27) | (b7 >> (64 - 27))) ^ b4;
    b6 += b5; b5 = ((b5 << 14) | (b5 >> (64 - 14))) ^ b6;
    b0 += b3; b3 = ((b3 << 42) | (b3 >> (64 - 42))) ^ b0;
    b4 += b1; b1 = ((b1 << 17) | (b1 >> (64 - 17))) ^ b4;
    b6 += b3; b3 = ((b3 << 49) | (b3 >> (64 - 49))) ^ b6;
    b0 += b5; b5 = ((b5 << 36) | (b5 >> (64 - 36))) ^ b0;
    b2 += b7; b7 = ((b7 << 39) | (b7 >> (64 - 39))) ^ b2;
    b6 += b1; b1 = ((b1 << 44) | (b1 >> (64 - 44))) ^ b6;
    b0 += b7; b7 = ((b7 << 9) | (b7 >> (64 - 9))) ^ b0;
    b2 += b5; b5 = ((b5 << 54) | (b5 >> (64 - 54))) ^ b2;
    b4 += b3; b3 = ((b3 << 56) | (b3 >> (64 - 56))) ^ b4;
    b1 += k0; b0 += b1 + k8; b1 = ((b1 << 39) | (b1 >> (64 - 39))) ^ b0;
    b3 += k2; b2 += b3 + k1; b3 = ((b3 << 30) | (b3 >> (64 - 30))) ^ b2;
    b5 += k4 + t2; b4 += b5 + k3; b5 = ((b5 << 34) | (b5 >> (64 - 34))) ^ b4;
    b7 += k6 + 17; b6 += b7 + k5 + t0; b7 = ((b7 << 24) | (b7 >> (64 - 24))) ^ b6;
    b2 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b2;
    b4 += b7; b7 = ((b7 << 50) | (b7 >> (64 - 50))) ^ b4;
    b6 += b5; b5 = ((b5 << 10) | (b5 >> (64 - 10))) ^ b6;
    b0 += b3; b3 = ((b3 << 17) | (b3 >> (64 - 17))) ^ b0;
    b4 += b1; b1 = ((b1 << 25) | (b1 >> (64 - 25))) ^ b4;
    b6 += b3; b3 = ((b3 << 29) | (b3 >> (64 - 29))) ^ b6;
    b0 += b5; b5 = ((b5 << 39) | (b5 >> (64 - 39))) ^ b0;
    b2 += b7; b7 = ((b7 << 43) | (b7 >> (64 - 43))) ^ b2;
    b6 += b1; b1 = ((b1 << 8) | (b1 >> (64 - 8))) ^ b6;
    b0 += b7; b7 = ((b7 << 35) | (b7 >> (64 - 35))) ^ b0;
    b2 += b5; b5 = ((b5 << 56) | (b5 >> (64 - 56))) ^ b2;
    b4 += b3; b3 = ((b3 << 22) | (b3 >> (64 - 22))) ^ b4;

    output[0] = b0 + k0;
    output[1] = b1 + k1;
    output[2] = b2 + k2;
    output[3] = b3 + k3;
    output[4] = b4 + k4;
    output[5] = b5 + k5 + t0;
    output[6] = b6 + k6 + t1;
    output[7] = b7 + k7 + 18;
}

func (tf *threefish512) decrypt(input, output [] uint64) {
    
    b0 := input[0]; b1 := input[1]
    b2 := input[2]; b3 := input[3]
    b4 := input[4]; b5 := input[5]
    b6 := input[6]; b7 := input[7]

    k0 := tf.expanedKey[0]; k1 := tf.expanedKey[1]
    k2 := tf.expanedKey[2]; k3 := tf.expanedKey[3]
    k4 := tf.expanedKey[4]; k5 := tf.expanedKey[5]
    k6 := tf.expanedKey[6]; k7 := tf.expanedKey[7]
    k8 := tf.expanedKey[8];
      
    t0 := tf.expanedTweak[0]
    t1 := tf.expanedTweak[1]
    t2 := tf.expanedTweak[2]

    var tmp uint64

    b0 -= k0;
    b1 -= k1;
    b2 -= k2;
    b3 -= k3;
    b4 -= k4;
    b5 -= k5 + t0;
    b6 -= k6 + t1;
    b7 -= k7 + 18;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k5 + t0; b7 -= k6 + 17;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k3; b5 -= k4 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k1; b3 -= k2;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k8; b1 -= k0;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k4 + t2; b7 -= k5 + 16;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k2; b5 -= k3 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k0; b3 -= k1;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k7; b1 -= k8;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k3 + t1; b7 -= k4 + 15;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k1; b5 -= k2 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k8; b3 -= k0;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k6; b1 -= k7;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k2 + t0; b7 -= k3 + 14;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k0; b5 -= k1 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k7; b3 -= k8;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k5; b1 -= k6;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k1 + t2; b7 -= k2 + 13;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k8; b5 -= k0 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k6; b3 -= k7;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k4; b1 -= k5;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k0 + t1; b7 -= k1 + 12;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k7; b5 -= k8 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k5; b3 -= k6;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k3; b1 -= k4;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k8 + t0; b7 -= k0 + 11;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k6; b5 -= k7 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k4; b3 -= k5;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k2; b1 -= k3;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k7 + t2; b7 -= k8 + 10;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k5; b5 -= k6 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k3; b3 -= k4;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k1; b1 -= k2;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k6 + t1; b7 -= k7 + 9;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k4; b5 -= k5 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k2; b3 -= k3;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k0; b1 -= k1;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k5 + t0; b7 -= k6 + 8;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k3; b5 -= k4 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k1; b3 -= k2;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k8; b1 -= k0;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k4 + t2; b7 -= k5 + 7;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k2; b5 -= k3 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k0; b3 -= k1;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k7; b1 -= k8;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k3 + t1; b7 -= k4 + 6;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k1; b5 -= k2 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k8; b3 -= k0;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k6; b1 -= k7;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k2 + t0; b7 -= k3 + 5;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k0; b5 -= k1 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k7; b3 -= k8;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k5; b1 -= k6;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k1 + t2; b7 -= k2 + 4;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k8; b5 -= k0 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k6; b3 -= k7;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k4; b1 -= k5;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k0 + t1; b7 -= k1 + 3;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k7; b5 -= k8 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k5; b3 -= k6;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k3; b1 -= k4;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k8 + t0; b7 -= k0 + 2;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k6; b5 -= k7 + t2;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k4; b3 -= k5;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k2; b1 -= k3;
    tmp = b3 ^ b4; b3 = (tmp >> 22) | (tmp << (64 - 22)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 56) | (tmp << (64 - 56)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 35) | (tmp << (64 - 35)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 8) | (tmp << (64 - 8)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 43) | (tmp << (64 - 43)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 29) | (tmp << (64 - 29)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 25) | (tmp << (64 - 25)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 17) | (tmp << (64 - 17)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 50) | (tmp << (64 - 50)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 24) | (tmp << (64 - 24)); b6 -= b7 + k7 + t2; b7 -= k8 + 1;
    tmp = b5 ^ b4; b5 = (tmp >> 34) | (tmp << (64 - 34)); b4 -= b5 + k5; b5 -= k6 + t1;
    tmp = b3 ^ b2; b3 = (tmp >> 30) | (tmp << (64 - 30)); b2 -= b3 + k3; b3 -= k4;
    tmp = b1 ^ b0; b1 = (tmp >> 39) | (tmp << (64 - 39)); b0 -= b1 + k1; b1 -= k2;
    tmp = b3 ^ b4; b3 = (tmp >> 56) | (tmp << (64 - 56)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 54) | (tmp << (64 - 54)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b7;
    tmp = b1 ^ b6; b1 = (tmp >> 44) | (tmp << (64 - 44)); b6 -= b1;
    tmp = b7 ^ b2; b7 = (tmp >> 39) | (tmp << (64 - 39)); b2 -= b7;
    tmp = b5 ^ b0; b5 = (tmp >> 36) | (tmp << (64 - 36)); b0 -= b5;
    tmp = b3 ^ b6; b3 = (tmp >> 49) | (tmp << (64 - 49)); b6 -= b3;
    tmp = b1 ^ b4; b1 = (tmp >> 17) | (tmp << (64 - 17)); b4 -= b1;
    tmp = b3 ^ b0; b3 = (tmp >> 42) | (tmp << (64 - 42)); b0 -= b3;
    tmp = b5 ^ b6; b5 = (tmp >> 14) | (tmp << (64 - 14)); b6 -= b5;
    tmp = b7 ^ b4; b7 = (tmp >> 27) | (tmp << (64 - 27)); b4 -= b7;
    tmp = b1 ^ b2; b1 = (tmp >> 33) | (tmp << (64 - 33)); b2 -= b1;
    tmp = b7 ^ b6; b7 = (tmp >> 37) | (tmp << (64 - 37)); b6 -= b7 + k6 + t1; b7 -= k7;
    tmp = b5 ^ b4; b5 = (tmp >> 19) | (tmp << (64 - 19)); b4 -= b5 + k4; b5 -= k5 + t0;
    tmp = b3 ^ b2; b3 = (tmp >> 36) | (tmp << (64 - 36)); b2 -= b3 + k2; b3 -= k3;
    tmp = b1 ^ b0; b1 = (tmp >> 46) | (tmp << (64 - 46)); b0 -= b1 + k0; b1 -= k1;

    output[7] = b7;
    output[6] = b6;
    output[5] = b5;
    output[4] = b4;
    output[3] = b3;
    output[2] = b2;
    output[1] = b1;
    output[0] = b0;
}

/*
 * Threefish 1024 implementation
 */
const CIPHER_SIZE_1024 = 1024
const CIPHER_QWORDS_1024 = CIPHER_SIZE_1024 / 64
const EXPANDED_KEY_SIZE_1024 = CIPHER_QWORDS_1024 + 1

type threefish1024 struct {
    expanedTweak [EXPANDED_TWEAK_SIZE]uint64
    expanedKey [EXPANDED_KEY_SIZE_1024]uint64
    tmpData1, tmpData2  [CIPHER_QWORDS_1024]uint64
}

// Get an initialized Threefish1024 structure
func newThreefish1024(key []byte, tweak []uint64) (*threefish1024, os.Error) {
    c := new(threefish1024)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_1024 - 1] = KEY_SCHEDULE_CONST

    // Convert key bytes to required uint64 array
    if key != nil {
        var tmpkey [EXPANDED_KEY_SIZE_1024]uint64        
        for i := 0; i < EXPANDED_KEY_SIZE_1024 - 1; i++ {
            tmpkey[i] = binary.LittleEndian.Uint64(key[i*8:i*8+8])
        }
        setKey(tmpkey[:], c.expanedKey[:]);
    }
    return c, nil
}

// Get an initialized Threefish1024 structure
func newThreefish1024_64(key, tweak []uint64) (*threefish1024, os.Error) {
    c := new(threefish1024)

    // Initialize tweak 
    setTweak(tweak, c.expanedTweak[:])

    c.expanedKey[EXPANDED_KEY_SIZE_1024 - 1] = KEY_SCHEDULE_CONST

    setKey(key, c.expanedKey[:]);
    return c, nil
}

func (tf *threefish1024) getTempData() ([]uint64, []uint64) {
    return tf.tmpData1[:], tf.tmpData2[:]
}

func (tf *threefish1024) setTweak(tweak []uint64) {
    setTweak(tweak, tf.expanedTweak[:])    
}

func (tf *threefish1024) setKey(key []uint64) {
    setKey(key, tf.expanedKey[:])    
}

func (tf *threefish1024) encrypt(input, output [] uint64) {
    
    b0 := input[0]; b1 := input[1]
    b2 := input[2]; b3 := input[3]
    b4 := input[4]; b5 := input[5]
    b6 := input[6]; b7 := input[7]
    b8 := input[8];   b9 := input[9]
    b10 := input[10]; b11 := input[11]
    b12 := input[12]; b13 := input[13]
    b14 := input[14]; b15 := input[15]

    k0 := tf.expanedKey[0]; k1 := tf.expanedKey[1]
    k2 := tf.expanedKey[2]; k3 := tf.expanedKey[3]
    k4 := tf.expanedKey[4]; k5 := tf.expanedKey[5]
    k6 := tf.expanedKey[6]; k7 := tf.expanedKey[7]
    
    k8 := tf.expanedKey[8];   k9 := tf.expanedKey[9]
    k10 := tf.expanedKey[10]; k11 := tf.expanedKey[11]
    k12 := tf.expanedKey[12]; k13 := tf.expanedKey[13]
    k14 := tf.expanedKey[14]; k15 := tf.expanedKey[15]
    k16 := tf.expanedKey[16];
      
    t0 := tf.expanedTweak[0]
    t1 := tf.expanedTweak[1]
    t2 := tf.expanedTweak[2]

    b1 += k1; b0 += b1 + k0; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k3; b2 += b3 + k2; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k5; b4 += b5 + k4; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k7; b6 += b7 + k6; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k9; b8 += b9 + k8; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k11; b10 += b11 + k10; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k13 + t0; b12 += b13 + k12; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k15; b14 += b15 + k14 + t1; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k2; b0 += b1 + k1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k4; b2 += b3 + k3; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k6; b4 += b5 + k5; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k8; b6 += b7 + k7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k10; b8 += b9 + k9; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k12; b10 += b11 + k11; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k14 + t1; b12 += b13 + k13; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k16 + 1; b14 += b15 + k15 + t2; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k3; b0 += b1 + k2; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k5; b2 += b3 + k4; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k7; b4 += b5 + k6; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k9; b6 += b7 + k8; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k11; b8 += b9 + k10; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k13; b10 += b11 + k12; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k15 + t2; b12 += b13 + k14; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k0 + 2; b14 += b15 + k16 + t0; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k4; b0 += b1 + k3; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k6; b2 += b3 + k5; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k8; b4 += b5 + k7; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k10; b6 += b7 + k9; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k12; b8 += b9 + k11; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k14; b10 += b11 + k13; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k16 + t0; b12 += b13 + k15; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k1 + 3; b14 += b15 + k0 + t1; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k5; b0 += b1 + k4; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k7; b2 += b3 + k6; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k9; b4 += b5 + k8; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k11; b6 += b7 + k10; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k13; b8 += b9 + k12; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k15; b10 += b11 + k14; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k0 + t1; b12 += b13 + k16; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k2 + 4; b14 += b15 + k1 + t2; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k6; b0 += b1 + k5; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k8; b2 += b3 + k7; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k10; b4 += b5 + k9; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k12; b6 += b7 + k11; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k14; b8 += b9 + k13; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k16; b10 += b11 + k15; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k1 + t2; b12 += b13 + k0; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k3 + 5; b14 += b15 + k2 + t0; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k7; b0 += b1 + k6; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k9; b2 += b3 + k8; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k11; b4 += b5 + k10; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k13; b6 += b7 + k12; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k15; b8 += b9 + k14; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k0; b10 += b11 + k16; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k2 + t0; b12 += b13 + k1; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k4 + 6; b14 += b15 + k3 + t1; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k8; b0 += b1 + k7; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k10; b2 += b3 + k9; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k12; b4 += b5 + k11; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k14; b6 += b7 + k13; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k16; b8 += b9 + k15; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k1; b10 += b11 + k0; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k3 + t1; b12 += b13 + k2; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k5 + 7; b14 += b15 + k4 + t2; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k9; b0 += b1 + k8; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k11; b2 += b3 + k10; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k13; b4 += b5 + k12; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k15; b6 += b7 + k14; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k0; b8 += b9 + k16; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k2; b10 += b11 + k1; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k4 + t2; b12 += b13 + k3; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k6 + 8; b14 += b15 + k5 + t0; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k10; b0 += b1 + k9; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k12; b2 += b3 + k11; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k14; b4 += b5 + k13; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k16; b6 += b7 + k15; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k1; b8 += b9 + k0; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k3; b10 += b11 + k2; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k5 + t0; b12 += b13 + k4; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k7 + 9; b14 += b15 + k6 + t1; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k11; b0 += b1 + k10; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k13; b2 += b3 + k12; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k15; b4 += b5 + k14; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k0; b6 += b7 + k16; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k2; b8 += b9 + k1; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k4; b10 += b11 + k3; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k6 + t1; b12 += b13 + k5; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k8 + 10; b14 += b15 + k7 + t2; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k12; b0 += b1 + k11; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k14; b2 += b3 + k13; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k16; b4 += b5 + k15; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k1; b6 += b7 + k0; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k3; b8 += b9 + k2; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k5; b10 += b11 + k4; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k7 + t2; b12 += b13 + k6; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k9 + 11; b14 += b15 + k8 + t0; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k13; b0 += b1 + k12; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k15; b2 += b3 + k14; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k0; b4 += b5 + k16; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k2; b6 += b7 + k1; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k4; b8 += b9 + k3; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k6; b10 += b11 + k5; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k8 + t0; b12 += b13 + k7; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k10 + 12; b14 += b15 + k9 + t1; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k14; b0 += b1 + k13; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k16; b2 += b3 + k15; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k1; b4 += b5 + k0; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k3; b6 += b7 + k2; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k5; b8 += b9 + k4; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k7; b10 += b11 + k6; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k9 + t1; b12 += b13 + k8; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k11 + 13; b14 += b15 + k10 + t2; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k15; b0 += b1 + k14; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k0; b2 += b3 + k16; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k2; b4 += b5 + k1; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k4; b6 += b7 + k3; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k6; b8 += b9 + k5; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k8; b10 += b11 + k7; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k10 + t2; b12 += b13 + k9; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k12 + 14; b14 += b15 + k11 + t0; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k16; b0 += b1 + k15; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k1; b2 += b3 + k0; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k3; b4 += b5 + k2; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k5; b6 += b7 + k4; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k7; b8 += b9 + k6; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k9; b10 += b11 + k8; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k11 + t0; b12 += b13 + k10; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k13 + 15; b14 += b15 + k12 + t1; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k0; b0 += b1 + k16; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k2; b2 += b3 + k1; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k4; b4 += b5 + k3; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k6; b6 += b7 + k5; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k8; b8 += b9 + k7; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k10; b10 += b11 + k9; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k12 + t1; b12 += b13 + k11; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k14 + 16; b14 += b15 + k13 + t2; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k1; b0 += b1 + k0; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k3; b2 += b3 + k2; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k5; b4 += b5 + k4; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k7; b6 += b7 + k6; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k9; b8 += b9 + k8; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k11; b10 += b11 + k10; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k13 + t2; b12 += b13 + k12; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k15 + 17; b14 += b15 + k14 + t0; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;
    b1 += k2; b0 += b1 + k1; b1 = ((b1 << 24) | (b1 >> (64 - 24))) ^ b0;
    b3 += k4; b2 += b3 + k3; b3 = ((b3 << 13) | (b3 >> (64 - 13))) ^ b2;
    b5 += k6; b4 += b5 + k5; b5 = ((b5 << 8) | (b5 >> (64 - 8))) ^ b4;
    b7 += k8; b6 += b7 + k7; b7 = ((b7 << 47) | (b7 >> (64 - 47))) ^ b6;
    b9 += k10; b8 += b9 + k9; b9 = ((b9 << 8) | (b9 >> (64 - 8))) ^ b8;
    b11 += k12; b10 += b11 + k11; b11 = ((b11 << 17) | (b11 >> (64 - 17))) ^ b10;
    b13 += k14 + t0; b12 += b13 + k13; b13 = ((b13 << 22) | (b13 >> (64 - 22))) ^ b12;
    b15 += k16 + 18; b14 += b15 + k15 + t1; b15 = ((b15 << 37) | (b15 >> (64 - 37))) ^ b14;
    b0 += b9; b9 = ((b9 << 38) | (b9 >> (64 - 38))) ^ b0;
    b2 += b13; b13 = ((b13 << 19) | (b13 >> (64 - 19))) ^ b2;
    b6 += b11; b11 = ((b11 << 10) | (b11 >> (64 - 10))) ^ b6;
    b4 += b15; b15 = ((b15 << 55) | (b15 >> (64 - 55))) ^ b4;
    b10 += b7; b7 = ((b7 << 49) | (b7 >> (64 - 49))) ^ b10;
    b12 += b3; b3 = ((b3 << 18) | (b3 >> (64 - 18))) ^ b12;
    b14 += b5; b5 = ((b5 << 23) | (b5 >> (64 - 23))) ^ b14;
    b8 += b1; b1 = ((b1 << 52) | (b1 >> (64 - 52))) ^ b8;
    b0 += b7; b7 = ((b7 << 33) | (b7 >> (64 - 33))) ^ b0;
    b2 += b5; b5 = ((b5 << 4) | (b5 >> (64 - 4))) ^ b2;
    b4 += b3; b3 = ((b3 << 51) | (b3 >> (64 - 51))) ^ b4;
    b6 += b1; b1 = ((b1 << 13) | (b1 >> (64 - 13))) ^ b6;
    b12 += b15; b15 = ((b15 << 34) | (b15 >> (64 - 34))) ^ b12;
    b14 += b13; b13 = ((b13 << 41) | (b13 >> (64 - 41))) ^ b14;
    b8 += b11; b11 = ((b11 << 59) | (b11 >> (64 - 59))) ^ b8;
    b10 += b9; b9 = ((b9 << 17) | (b9 >> (64 - 17))) ^ b10;
    b0 += b15; b15 = ((b15 << 5) | (b15 >> (64 - 5))) ^ b0;
    b2 += b11; b11 = ((b11 << 20) | (b11 >> (64 - 20))) ^ b2;
    b6 += b13; b13 = ((b13 << 48) | (b13 >> (64 - 48))) ^ b6;
    b4 += b9; b9 = ((b9 << 41) | (b9 >> (64 - 41))) ^ b4;
    b14 += b1; b1 = ((b1 << 47) | (b1 >> (64 - 47))) ^ b14;
    b8 += b5; b5 = ((b5 << 28) | (b5 >> (64 - 28))) ^ b8;
    b10 += b3; b3 = ((b3 << 16) | (b3 >> (64 - 16))) ^ b10;
    b12 += b7; b7 = ((b7 << 25) | (b7 >> (64 - 25))) ^ b12;
    b1 += k3; b0 += b1 + k2; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b0;
    b3 += k5; b2 += b3 + k4; b3 = ((b3 << 9) | (b3 >> (64 - 9))) ^ b2;
    b5 += k7; b4 += b5 + k6; b5 = ((b5 << 37) | (b5 >> (64 - 37))) ^ b4;
    b7 += k9; b6 += b7 + k8; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b6;
    b9 += k11; b8 += b9 + k10; b9 = ((b9 << 12) | (b9 >> (64 - 12))) ^ b8;
    b11 += k13; b10 += b11 + k12; b11 = ((b11 << 47) | (b11 >> (64 - 47))) ^ b10;
    b13 += k15 + t1; b12 += b13 + k14; b13 = ((b13 << 44) | (b13 >> (64 - 44))) ^ b12;
    b15 += k0 + 19; b14 += b15 + k16 + t2; b15 = ((b15 << 30) | (b15 >> (64 - 30))) ^ b14;
    b0 += b9; b9 = ((b9 << 16) | (b9 >> (64 - 16))) ^ b0;
    b2 += b13; b13 = ((b13 << 34) | (b13 >> (64 - 34))) ^ b2;
    b6 += b11; b11 = ((b11 << 56) | (b11 >> (64 - 56))) ^ b6;
    b4 += b15; b15 = ((b15 << 51) | (b15 >> (64 - 51))) ^ b4;
    b10 += b7; b7 = ((b7 << 4) | (b7 >> (64 - 4))) ^ b10;
    b12 += b3; b3 = ((b3 << 53) | (b3 >> (64 - 53))) ^ b12;
    b14 += b5; b5 = ((b5 << 42) | (b5 >> (64 - 42))) ^ b14;
    b8 += b1; b1 = ((b1 << 41) | (b1 >> (64 - 41))) ^ b8;
    b0 += b7; b7 = ((b7 << 31) | (b7 >> (64 - 31))) ^ b0;
    b2 += b5; b5 = ((b5 << 44) | (b5 >> (64 - 44))) ^ b2;
    b4 += b3; b3 = ((b3 << 47) | (b3 >> (64 - 47))) ^ b4;
    b6 += b1; b1 = ((b1 << 46) | (b1 >> (64 - 46))) ^ b6;
    b12 += b15; b15 = ((b15 << 19) | (b15 >> (64 - 19))) ^ b12;
    b14 += b13; b13 = ((b13 << 42) | (b13 >> (64 - 42))) ^ b14;
    b8 += b11; b11 = ((b11 << 44) | (b11 >> (64 - 44))) ^ b8;
    b10 += b9; b9 = ((b9 << 25) | (b9 >> (64 - 25))) ^ b10;
    b0 += b15; b15 = ((b15 << 9) | (b15 >> (64 - 9))) ^ b0;
    b2 += b11; b11 = ((b11 << 48) | (b11 >> (64 - 48))) ^ b2;
    b6 += b13; b13 = ((b13 << 35) | (b13 >> (64 - 35))) ^ b6;
    b4 += b9; b9 = ((b9 << 52) | (b9 >> (64 - 52))) ^ b4;
    b14 += b1; b1 = ((b1 << 23) | (b1 >> (64 - 23))) ^ b14;
    b8 += b5; b5 = ((b5 << 31) | (b5 >> (64 - 31))) ^ b8;
    b10 += b3; b3 = ((b3 << 37) | (b3 >> (64 - 37))) ^ b10;
    b12 += b7; b7 = ((b7 << 20) | (b7 >> (64 - 20))) ^ b12;


    output[0] = b0 + k3;
    output[1] = b1 + k4;
    output[2] = b2 + k5;
    output[3] = b3 + k6;
    output[4] = b4 + k7;
    output[5] = b5 + k8;
    output[6] = b6 + k9;
    output[7] = b7 + k10;
    output[8] = b8 + k11;
    output[9] = b9 + k12;
    output[10] = b10 + k13;
    output[11] = b11 + k14;
    output[12] = b12 + k15;
    output[13] = b13 + k16 + t2;
    output[14] = b14 + k0 + t0;
    output[15] = b15 + k1 + 20;
}

func (tf *threefish1024) decrypt(input, output [] uint64) {
    
    b0 := input[0]; b1 := input[1]
    b2 := input[2]; b3 := input[3]
    b4 := input[4]; b5 := input[5]
    b6 := input[6]; b7 := input[7]
    b8 := input[8];   b9 := input[9]
    b10 := input[10]; b11 := input[11]
    b12 := input[12]; b13 := input[13]
    b14 := input[14]; b15 := input[15]

    k0 := tf.expanedKey[0]; k1 := tf.expanedKey[1]
    k2 := tf.expanedKey[2]; k3 := tf.expanedKey[3]
    k4 := tf.expanedKey[4]; k5 := tf.expanedKey[5]
    k6 := tf.expanedKey[6]; k7 := tf.expanedKey[7]
    
    k8 := tf.expanedKey[8];   k9 := tf.expanedKey[9]
    k10 := tf.expanedKey[10]; k11 := tf.expanedKey[11]
    k12 := tf.expanedKey[12]; k13 := tf.expanedKey[13]
    k14 := tf.expanedKey[14]; k15 := tf.expanedKey[15]
    k16 := tf.expanedKey[16];
      
    t0 := tf.expanedTweak[0]
    t1 := tf.expanedTweak[1]
    t2 := tf.expanedTweak[2]

    var tmp uint64

    b0 -= k3;
    b1 -= k4;
    b2 -= k5;
    b3 -= k6;
    b4 -= k7;
    b5 -= k8;
    b6 -= k9;
    b7 -= k10;
    b8 -= k11;
    b9 -= k12;
    b10 -= k13;
    b11 -= k14;
    b12 -= k15;
    b13 -= k16 + t2;
    b14 -= k0 + t0;
    b15 -= k1 + 20;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k16 + t2; b15 -= k0 + 19;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k14; b13 -= k15 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k12; b11 -= k13;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k10; b9 -= k11;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k8; b7 -= k9;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k6; b5 -= k7;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k4; b3 -= k5;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k2; b1 -= k3;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k15 + t1; b15 -= k16 + 18;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k13; b13 -= k14 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k11; b11 -= k12;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k9; b9 -= k10;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k7; b7 -= k8;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k5; b5 -= k6;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k3; b3 -= k4;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k1; b1 -= k2;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k14 + t0; b15 -= k15 + 17;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k12; b13 -= k13 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k10; b11 -= k11;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k8; b9 -= k9;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k6; b7 -= k7;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k4; b5 -= k5;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k2; b3 -= k3;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k0; b1 -= k1;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k13 + t2; b15 -= k14 + 16;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k11; b13 -= k12 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k9; b11 -= k10;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k7; b9 -= k8;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k5; b7 -= k6;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k3; b5 -= k4;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k1; b3 -= k2;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k16; b1 -= k0;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k12 + t1; b15 -= k13 + 15;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k10; b13 -= k11 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k8; b11 -= k9;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k6; b9 -= k7;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k4; b7 -= k5;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k2; b5 -= k3;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k0; b3 -= k1;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k15; b1 -= k16;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k11 + t0; b15 -= k12 + 14;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k9; b13 -= k10 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k7; b11 -= k8;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k5; b9 -= k6;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k3; b7 -= k4;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k1; b5 -= k2;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k16; b3 -= k0;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k14; b1 -= k15;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k10 + t2; b15 -= k11 + 13;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k8; b13 -= k9 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k6; b11 -= k7;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k4; b9 -= k5;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k2; b7 -= k3;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k0; b5 -= k1;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k15; b3 -= k16;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k13; b1 -= k14;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k9 + t1; b15 -= k10 + 12;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k7; b13 -= k8 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k5; b11 -= k6;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k3; b9 -= k4;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k1; b7 -= k2;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k16; b5 -= k0;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k14; b3 -= k15;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k12; b1 -= k13;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k8 + t0; b15 -= k9 + 11;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k6; b13 -= k7 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k4; b11 -= k5;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k2; b9 -= k3;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k0; b7 -= k1;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k15; b5 -= k16;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k13; b3 -= k14;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k11; b1 -= k12;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k7 + t2; b15 -= k8 + 10;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k5; b13 -= k6 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k3; b11 -= k4;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k1; b9 -= k2;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k16; b7 -= k0;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k14; b5 -= k15;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k12; b3 -= k13;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k10; b1 -= k11;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k6 + t1; b15 -= k7 + 9;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k4; b13 -= k5 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k2; b11 -= k3;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k0; b9 -= k1;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k15; b7 -= k16;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k13; b5 -= k14;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k11; b3 -= k12;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k9; b1 -= k10;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k5 + t0; b15 -= k6 + 8;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k3; b13 -= k4 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k1; b11 -= k2;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k16; b9 -= k0;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k14; b7 -= k15;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k12; b5 -= k13;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k10; b3 -= k11;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k8; b1 -= k9;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k4 + t2; b15 -= k5 + 7;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k2; b13 -= k3 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k0; b11 -= k1;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k15; b9 -= k16;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k13; b7 -= k14;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k11; b5 -= k12;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k9; b3 -= k10;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k7; b1 -= k8;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k3 + t1; b15 -= k4 + 6;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k1; b13 -= k2 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k16; b11 -= k0;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k14; b9 -= k15;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k12; b7 -= k13;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k10; b5 -= k11;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k8; b3 -= k9;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k6; b1 -= k7;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k2 + t0; b15 -= k3 + 5;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k0; b13 -= k1 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k15; b11 -= k16;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k13; b9 -= k14;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k11; b7 -= k12;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k9; b5 -= k10;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k7; b3 -= k8;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k5; b1 -= k6;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k1 + t2; b15 -= k2 + 4;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k16; b13 -= k0 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k14; b11 -= k15;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k12; b9 -= k13;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k10; b7 -= k11;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k8; b5 -= k9;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k6; b3 -= k7;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k4; b1 -= k5;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k0 + t1; b15 -= k1 + 3;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k15; b13 -= k16 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k13; b11 -= k14;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k11; b9 -= k12;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k9; b7 -= k10;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k7; b5 -= k8;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k5; b3 -= k6;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k3; b1 -= k4;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k16 + t0; b15 -= k0 + 2;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k14; b13 -= k15 + t2;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k12; b11 -= k13;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k10; b9 -= k11;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k8; b7 -= k9;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k6; b5 -= k7;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k4; b3 -= k5;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k2; b1 -= k3;
    tmp = b7 ^ b12; b7 = (tmp >> 20) | (tmp << (64 - 20)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 37) | (tmp << (64 - 37)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 31) | (tmp << (64 - 31)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 52) | (tmp << (64 - 52)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 35) | (tmp << (64 - 35)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 48) | (tmp << (64 - 48)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 9) | (tmp << (64 - 9)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 25) | (tmp << (64 - 25)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 44) | (tmp << (64 - 44)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 19) | (tmp << (64 - 19)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 46) | (tmp << (64 - 46)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 47) | (tmp << (64 - 47)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 44) | (tmp << (64 - 44)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 31) | (tmp << (64 - 31)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 41) | (tmp << (64 - 41)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 42) | (tmp << (64 - 42)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 53) | (tmp << (64 - 53)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 4) | (tmp << (64 - 4)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 56) | (tmp << (64 - 56)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 34) | (tmp << (64 - 34)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 16) | (tmp << (64 - 16)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 30) | (tmp << (64 - 30)); b14 -= b15 + k15 + t2; b15 -= k16 + 1;
    tmp = b13 ^ b12; b13 = (tmp >> 44) | (tmp << (64 - 44)); b12 -= b13 + k13; b13 -= k14 + t1;
    tmp = b11 ^ b10; b11 = (tmp >> 47) | (tmp << (64 - 47)); b10 -= b11 + k11; b11 -= k12;
    tmp = b9 ^ b8; b9 = (tmp >> 12) | (tmp << (64 - 12)); b8 -= b9 + k9; b9 -= k10;
    tmp = b7 ^ b6; b7 = (tmp >> 31) | (tmp << (64 - 31)); b6 -= b7 + k7; b7 -= k8;
    tmp = b5 ^ b4; b5 = (tmp >> 37) | (tmp << (64 - 37)); b4 -= b5 + k5; b5 -= k6;
    tmp = b3 ^ b2; b3 = (tmp >> 9) | (tmp << (64 - 9)); b2 -= b3 + k3; b3 -= k4;
    tmp = b1 ^ b0; b1 = (tmp >> 41) | (tmp << (64 - 41)); b0 -= b1 + k1; b1 -= k2;
    tmp = b7 ^ b12; b7 = (tmp >> 25) | (tmp << (64 - 25)); b12 -= b7;
    tmp = b3 ^ b10; b3 = (tmp >> 16) | (tmp << (64 - 16)); b10 -= b3;
    tmp = b5 ^ b8; b5 = (tmp >> 28) | (tmp << (64 - 28)); b8 -= b5;
    tmp = b1 ^ b14; b1 = (tmp >> 47) | (tmp << (64 - 47)); b14 -= b1;
    tmp = b9 ^ b4; b9 = (tmp >> 41) | (tmp << (64 - 41)); b4 -= b9;
    tmp = b13 ^ b6; b13 = (tmp >> 48) | (tmp << (64 - 48)); b6 -= b13;
    tmp = b11 ^ b2; b11 = (tmp >> 20) | (tmp << (64 - 20)); b2 -= b11;
    tmp = b15 ^ b0; b15 = (tmp >> 5) | (tmp << (64 - 5)); b0 -= b15;
    tmp = b9 ^ b10; b9 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b9;
    tmp = b11 ^ b8; b11 = (tmp >> 59) | (tmp << (64 - 59)); b8 -= b11;
    tmp = b13 ^ b14; b13 = (tmp >> 41) | (tmp << (64 - 41)); b14 -= b13;
    tmp = b15 ^ b12; b15 = (tmp >> 34) | (tmp << (64 - 34)); b12 -= b15;
    tmp = b1 ^ b6; b1 = (tmp >> 13) | (tmp << (64 - 13)); b6 -= b1;
    tmp = b3 ^ b4; b3 = (tmp >> 51) | (tmp << (64 - 51)); b4 -= b3;
    tmp = b5 ^ b2; b5 = (tmp >> 4) | (tmp << (64 - 4)); b2 -= b5;
    tmp = b7 ^ b0; b7 = (tmp >> 33) | (tmp << (64 - 33)); b0 -= b7;
    tmp = b1 ^ b8; b1 = (tmp >> 52) | (tmp << (64 - 52)); b8 -= b1;
    tmp = b5 ^ b14; b5 = (tmp >> 23) | (tmp << (64 - 23)); b14 -= b5;
    tmp = b3 ^ b12; b3 = (tmp >> 18) | (tmp << (64 - 18)); b12 -= b3;
    tmp = b7 ^ b10; b7 = (tmp >> 49) | (tmp << (64 - 49)); b10 -= b7;
    tmp = b15 ^ b4; b15 = (tmp >> 55) | (tmp << (64 - 55)); b4 -= b15;
    tmp = b11 ^ b6; b11 = (tmp >> 10) | (tmp << (64 - 10)); b6 -= b11;
    tmp = b13 ^ b2; b13 = (tmp >> 19) | (tmp << (64 - 19)); b2 -= b13;
    tmp = b9 ^ b0; b9 = (tmp >> 38) | (tmp << (64 - 38)); b0 -= b9;
    tmp = b15 ^ b14; b15 = (tmp >> 37) | (tmp << (64 - 37)); b14 -= b15 + k14 + t1; b15 -= k15;
    tmp = b13 ^ b12; b13 = (tmp >> 22) | (tmp << (64 - 22)); b12 -= b13 + k12; b13 -= k13 + t0;
    tmp = b11 ^ b10; b11 = (tmp >> 17) | (tmp << (64 - 17)); b10 -= b11 + k10; b11 -= k11;
    tmp = b9 ^ b8; b9 = (tmp >> 8) | (tmp << (64 - 8)); b8 -= b9 + k8; b9 -= k9;
    tmp = b7 ^ b6; b7 = (tmp >> 47) | (tmp << (64 - 47)); b6 -= b7 + k6; b7 -= k7;
    tmp = b5 ^ b4; b5 = (tmp >> 8) | (tmp << (64 - 8)); b4 -= b5 + k4; b5 -= k5;
    tmp = b3 ^ b2; b3 = (tmp >> 13) | (tmp << (64 - 13)); b2 -= b3 + k2; b3 -= k3;
    tmp = b1 ^ b0; b1 = (tmp >> 24) | (tmp << (64 - 24)); b0 -= b1 + k0; b1 -= k1;

    output[15] = b15;
    output[14] = b14;
    output[13] = b13;
    output[12] = b12;
    output[11] = b11;
    output[10] = b10;
    output[9] = b9;
    output[8] = b8;
    output[7] = b7;
    output[6] = b6;
    output[5] = b5;
    output[4] = b4;
    output[3] = b3;
    output[2] = b2;
    output[1] = b1;
    output[0] = b0;
}
