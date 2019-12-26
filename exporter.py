from prometheus_client import start_http_server, Summary
from prometheus_client import Counter
from prometheus_client import Gauge
from prometheus_client import Enum
from prometheus_client import Info
import random
import time
import sys
import urllib3
import json
import os
import logging

import dns.resolver
from pprint import pprint

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s', datefmt='%d-%b-%y %H:%M:%S')

ERROR_MAPPING = {'NXDOMAIN': 0, 'Found' : 1, 'NoAnswer': 2, 'NoNameservers': 3, 'Timeout': 4, 'Unknown': 5}

i_dnsrbl = Info('dnsrbl', 'General info')
e_dnsrbl_task_state = Enum('dnsrbl_task_state', 'Task state', states=['running', 'sleeping'])
g_dnsrbl_list_size =  Gauge('dnsrbl_list_size', 'number of blacklists active')

g_dnsrbl_query = Counter('dnsrbl_query', 'dns queries', ['list', 'ip', 'result'])
g_dnsrbl_status =  Gauge('dnsrbl_status', 'dnsrbl check status 0 = ok, 1 = found in blacklist, 2-5 = error', ['list', 'ip'])

g_httpbl_last_activity = Gauge('httpbl_last_activity', 'projecthoneypot.org last activity', ['list', 'ip'])
g_httpbl_thread_score = Gauge('httpbl_thread_score', 'projecthoneypot.org thread score', ['list', 'ip'])
g_httpbl_visitor_type = Gauge('httpbl_visitor_type', 'projecthoneypot.org visitor type', ['list', 'ip'])

# Create a metric to track time spent and requests made.
REQUEST_TIME = Summary('request_processing_seconds', 'Time spent processing request')

# Decorate function with metric.
@REQUEST_TIME.time()
def check_dnsrbl(ip, blacklist):
    """A dummy function that takes some time."""
    reverse_ip = convert_to_reverse_ip(ip)
    query = '{}.{}.'.format(reverse_ip,blacklist)
    if blacklist == 'dnsbl.httpbl.org':
        if get_httpbl_access_key() == None:
            logging.warning("skipping blacklist {} due to missing env DNSRBL_HTTP_BL_ACCESS_KEY".format(blacklist))
            return
        query = '{}.{}.{}.'.format(get_httpbl_access_key(), reverse_ip, blacklist)

    logging.info("checking {}.{}.".format(reverse_ip,blacklist))
    try:
        answers = dns.resolver.query(query, 'A')
        for rdata in answers:
            result = str(rdata)
            logging.warning("match: {} found in {}".format(result, blacklist))
            if blacklist == 'dnsbl.httpbl.org':
                dummy,last_activity,threat_score,visitor_type = result.split('.')
                g_httpbl_last_activity.labels(list=blacklist,ip=ip).set(last_activity)
                g_httpbl_thread_score.labels(list=blacklist,ip=ip).set(threat_score)
                g_httpbl_visitor_type.labels(list=blacklist,ip=ip).set(visitor_type)
                logging.warning("last_activity: {} days ago".format(last_activity))
                logging.warning("thread_score: {}".format(threat_score))
                logging.warning("visitor_type: {}".format(visitor_type))
        g_dnsrbl_query.labels(list=blacklist,ip=ip,result='Found').inc()
        g_dnsrbl_status.labels(list=blacklist,ip=ip).set(ERROR_MAPPING['Found'])

    except (dns.resolver.NXDOMAIN, dns.resolver.NoAnswer, dns.resolver.NoNameservers, dns.resolver.Timeout, ) as error:
        logging.info("error: {}".format(error.__class__.__name__))
        g_dnsrbl_query.labels(list=blacklist,ip=ip,result=error.__class__.__name__).inc()
        g_dnsrbl_status.labels(list=blacklist,ip=ip).set(ERROR_MAPPING[error.__class__.__name__])

    except Exception as e:
        logging.error("Exception occurred", exc_info=True)
        g_dnsrbl_query.labels(list=blacklist,ip=ip,result='Unknown').inc()
        g_dnsrbl_status.labels(list=blacklist,ip=ip).set(ERROR_MAPPING['Unknown'])

def convert_to_reverse_ip(ip):
    """A dummy function that takes some time."""
    reverse_ip = ip.split('.')
    reverse_ip.reverse()
    reverse_ip = '.'.join(reverse_ip)
    return reverse_ip

def get_external_ip():
    """A dummy function that takes some time."""
    http = urllib3.PoolManager()

    #r = http.request('GET', 'http://ifconfig.co/json')
    #result = json.loads(r.data.decode('utf-8'))
    #return result['ip']

    r = http.request('GET', 'http://ifconfig.me')
    return r.data.decode('utf-8')

def get_static_ip():
    return os.getenv('DNSRBL_CHECK_IP')

def get_check_ip_mode():
    if os.getenv('DNSRBL_CHECK_IP', None)  != None:
        return 'static'
    return 'dynamic'

def get_lists():
    if os.getenv('DNSRBL_LISTS', None)  != None:
        dnsrbl_lists = os.getenv('DNSRBL_LISTS')
        dnsrbl_lists = dnsrbl_lists.split(' ')
        return dnsrbl_lists

    with open(os.getenv('DNSRBL_LISTS_FILENAME', "lists.txt")) as f:
        dnsrbl_lists = f.readlines()
    dnsrbl_lists = [x.strip() for x in dnsrbl_lists]
    return dnsrbl_lists

def get_httpbl_access_key():
    return os.getenv('DNSRBL_HTTP_BL_ACCESS_KEY', None)

def get_delay_between_requests():
    return int(os.getenv('DNSRBL_DELAY_REQUESTS', 1))

def get_delay_between_runs():
    return int(os.getenv('DNSRBL_DELAY_RUNS', 60))

def get_lisener_port():
    return int(os.getenv('DNSRBL_PORT', 8000))

if __name__ == '__main__':
    # Start up the server to expose the metrics.
    start_http_server(get_lisener_port())
    # Generate some requests.
    while True:
        if (get_check_ip_mode() == 'dynamic'):
            check_ip = get_external_ip()
        else:
            check_ip = get_static_ip()
        logging.info("Using {} as {} check ip".format(check_ip, get_check_ip_mode()))
        dnsrbl_lists = get_lists()
        g_dnsrbl_list_size.set(len(dnsrbl_lists))
        logging.info("Using {} blacklists".format(len(dnsrbl_lists)))
        i_dnsrbl.info(
            {
                'check_ip' : check_ip,
                'check_ip_mode' : get_check_ip_mode(),
                'delay_between_requests' : '{}s'.format(get_delay_between_requests()),
                'delay_between_runs' : '{}s'.format(get_delay_between_runs())
            }
        )

        for blacklist in dnsrbl_lists:
            if blacklist.startswith( '#' ):
                continue
            e_dnsrbl_task_state.state('running')
            check_dnsrbl(check_ip, blacklist)
            e_dnsrbl_task_state.state('sleeping')
            logging.debug("sleeping...")
            time.sleep( get_delay_between_requests() )
        logging.debug("sleeping...")
        time.sleep( get_delay_between_runs() )