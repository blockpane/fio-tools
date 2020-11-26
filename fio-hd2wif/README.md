# fio-hd2wif

Simple utility that derives a WIF from a HD nmemonic phrase, or generates a random HD phrase and prints a WIF.

This utility uses the reserved path in [SLIP44](https://github.com/satoshilabs/slips/blob/master/slip-0044.md) for the FIO chain: `44'/235'/0`

```
Usage of fio-hd2wif:
  -f string
        optional: Read HD mnemonic from file
  -i int
        optional: which key index to derive, default 0
  -n    Generate a new HD phrase and print the first WIF
  -w int
        number of words for new nmemonic, valid values are: 12, 15, 18, 21, or 24 (default 24)
```

### Examples

Get the first key from a phase, entered from STDIN:

```
$ fio-hd2wif
Please enter the mnemonic phrase: tree salt elite boss wide blade involve saddle faculty citizen deer crater action possible trophy scissors sudden rigid surge system position silk odor science
5JmxdCbDiWSSHEzc9REaKNty78kUn4DU5PH38MWmtYvMY5hw38o
(index 0)

```

Get the 4th WIF from the index (position 3)
```
$ fio-hd2wif -i 3
Please enter the mnemonic phrase: tree salt elite boss wide blade involve saddle faculty citizen deer crater action possible trophy scissors sudden rigid surge system position silk odor science
5JhALXxxWsYUwPar2JdsVamExmFex15dyu7FADnLMUfHYmL6Nay
(index 3)

```

Read a HD nmemonic from a file and print the first WIF:
```
$ fio-hd2wif -f local-phrase.txt
5JmxdCbDiWSSHEzc9REaKNty78kUn4DU5PH38MWmtYvMY5hw38o
(index 0)

```

Generate a random nmemonic and print the first WIF:
```
$ fio-hd2wif -n
decline asset blood nature green else replace witness couple cement please mesh drastic electric appear curious drink differ oven doctor parent input evil enemy
5JcATMabD46jrJ8TnNFmBxsY95bgSiP4FMQ3Pp3MxhximDPmrcR
(index 0)

```

Generate a random 21 word nmemonic and print the first WIF:
```
$ fio-hd2wif -n -w 21
tornado suggest broken deal nasty evoke sister rocket middle demand casual school maple whale since enough chaos penalty upon ticket donor
5KeQX3i2FCPdbbyYFKq9ZMaN93zgo72FaGueCouJmXoT8UyHyJk
(index 0)

```
