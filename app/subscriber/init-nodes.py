#!/usr/bin/env python

import argparse
import os
import shutil
import subprocess
import toml

# This function checks if the IP 127.0.0.i is assigned to the loopback interface
def check_lo_address( i ):
    address = "127.0.0.{}".format( i )
    command = "ip addr show lo"
    output = subprocess.run(
        command,
        shell = True,
        capture_output = True,
        check = True,
        text = True,
    ).stdout
    return address in output

# This function parses the arguments and validates them
def parse_args():
    parser = argparse.ArgumentParser(
        description = "Initialize a localnet with multiple nodes",
    )
    parser.add_argument(
        "-b", "--binary",
        type = str,
        default = "exocored",
        help = "The binary path (or just the name if it is in the PATH)",
        required = True,
    )
    parser.add_argument(
        "-c", "--chain-id",
        type = str,
        default = "exocoretestnet_233-6",
        help = "The Chain ID",
        required = True,
    )
    parser.add_argument(
        "-l", "--log-level",
        type = str,
        default = "info",
        help = "The log level",
        required = True,
    )
    home = os.environ[ "HOME" ]
    parser.add_argument(
        "-f", "--folder",
        type = str,
        default = "{}/.tmp-exocored/node".format( home ),
        help = "The folder path for the nodes. It will be suffixed by the node number",
        required = True,
    )
    parser.add_argument(
        "-m", "--mnemonics-file",
        type = str,
        default = "mnemonics.txt",
        help = "The file containing the mnemonics",
        required = True,
    )
    parser.add_argument(
        "-d", "--denom",
        type = str,
        help = "The denomination used during the init",
        required = False,
    )
    parser.add_argument(
        "-p", "--port-offset",
        type = int,
        default = 0,
        help = "The port offset for the nodes",
        required = False,
    )
    args = parser.parse_args()
    # now validate
    if not os.path.exists( args.binary ):
        raise ValueError( "The binary does not exist" )
    # check executable
    if not os.access( args.binary, os.X_OK ):
        raise ValueError( "The binary is not executable" )
    # parent folder must exist
    parent_folder = os.path.dirname( args.folder )
    if not os.path.exists( parent_folder ):
        os.mkdir( parent_folder )
    if not os.path.exists( args.mnemonics_file ):
        raise ValueError( "The mnemonics file does not exist" )
    if not os.path.isfile( args.mnemonics_file ):
        raise ValueError( "The mnemonics file is not a file" )
    # let log level and chain-id be validated by Cosmos SDK
    return args

# This function recursively replaces the IP addresses in the TOML files
def recursive_replace( data, ip, port_offset ):
    if isinstance( data, dict ):
        for key, value in data.items():
            if isinstance( value, str ):
                replaced = False
                for check in [
                    "127.0.0.1", "localhost", "0.0.0.0"
                ]:
                    if check in value:
                        data[ key ] = value.replace( check, ip )
                        replaced = True
                        break
                if not replaced and value.startswith( ":" ) and value[ 1: ].isnumeric():
                    data[ key ] = "{}:{}".format( ip, value[ 1: ] )
                    replaced = True
                if replaced:
                    port = data[ key ].split( ":" )[ -1 ]
                    data[ key ] = data[ key ].replace( str( port ), str( int( port ) + port_offset ) )
            else:
                recursive_replace( value, ip, port_offset )
    elif isinstance( data, list ):
        for i in range( len( data ) ):
            if isinstance( data[ i ], str ):
                data[ i ] = data[ i ].replace( "asd", "asd" )
            else:
                recursive_replace( data[ i ], ip, port_offset )

def main():
    args = parse_args()
    mnemonics = open( args.mnemonics_file, "r" ).read().split( "\n" )
    n = len( mnemonics )
    if n == 0:
        raise ValueError( "No mnemonics found in the file" )
    for i in range( n ):
        # other validation such as BIP-39 mnemonic length is left to the Cosmos SDK
        if len( mnemonics[ i ] ) == 0:
            raise ValueError( "Empty mnemonic found at line {}".format( i + 1 ) )
        if not check_lo_address( i + 1 ):
            raise ValueError( "The IP 127.0.0.{} is not assigned to the loopback interface".format( i + 1 ) )
    folders = []
    peer_ids = []
    for i in range( n ):
        moniker = "node{}".format( i + 1 )
        folder = "{}{}".format( args.folder, i + 1 )
        if os.path.exists( folder ):
            print( "Deleting existing folder", folder )
            shutil.rmtree( folder )
        if args.denom:
            subprocess.run(
                'echo "{}" | {} init {} --chain-id {} --recover --home {} --default-denom {}'.format(
                    mnemonics[ i ], args.binary, moniker, args.chain_id, folder, args.denom,
                ),
                shell = True, text = True,
                capture_output = True,
                check = True,
            )
        else:
            subprocess.run(
                'echo "{}" | {} init {} --chain-id {} --recover --home {}'.format(
                    args.mnemonics[ i ], args.binary, moniker, args.chain_id, folder,
                ),
                shell = True, text = True,
                capture_output = True,
                check = True,
            )
        folders.append( folder )
        # get the peer id
        peer_id = subprocess.run(
            [
                args.binary,
                "tendermint",
                "show-node-id",
                "--home",
                folder,
            ],
            capture_output = True,
            check = True,
        ).stdout.decode().strip()
        peer_ids.append( peer_id )
    # now it's time to do the configuration. it should be done after all inits so that
    # persistent peers can be set up correctly.
    for i in range( n ):
        folder = folders[ i ]
        config_file_path = "{}/config/config.toml".format( folder )
        with open( config_file_path, "r" ) as f:
            data = toml.load( f )
        recursive_replace( data, "127.0.0.{}".format( i + 1 ), args.port_offset )
        data[ "log_level" ] = args.log_level
        # So that 127.0.0.x when it goes via NAT is not considered a duplicate of 127.0.0.1
        data[ "p2p" ][ "allow_duplicate_ip" ] = True
        # Since we have only persistent peers, we can set this to false
        data[ "p2p" ][ "pex" ] = False
        # Allow non-routable addresses (RFC1918 and friends) to be considered
        data[ "p2p" ][ "addr_book_strict" ] = False
        # Now configure persistent peers
        persistent_peers = []
        for j in range( n ):
            if i == j:
                continue
            persistent_peers.append(
                "{}@{}:{}".format(
                    peer_ids[ j ],
                    "127.0.0.{}".format( j + 1 ),
                    26656 + args.port_offset,
                )
            )
        data[ "p2p" ][ "persistent_peers" ] = ",".join( persistent_peers )
        with open( config_file_path, "w" ) as file:
            toml.dump( data, file )
        # grpc and grpc-web bindings
        app_file_path = "{}/config/app.toml".format( folder )
        with open( app_file_path, "r" ) as f:
            app_data = toml.load( f )
        recursive_replace( app_data, "127.0.0.{}".format( i + 1 ), args.port_offset )
        with open( app_file_path, "w" ) as file:
            toml.dump( app_data, file )
    for i in range( n ):
        print( "done, run: `{} start --home {}`".format( args.binary, folders[ i ] ) )

if __name__ == "__main__":
    main()