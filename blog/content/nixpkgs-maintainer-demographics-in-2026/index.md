+++
title = "Nixpkgs maintainer demographics in 2026"
description = "What can we learn from nixpkgs maintainers' emails, and what can they tell us about the Nix ecosystem?"
date = 2026-11-20
+++

I recently stumbled upon this article, "[Nix maintainer demographics](https://kyswtn.com/posts/nix-maintainers)", which
tried to find patterns in the email addresses used by nixpkgs maintainers to see which email providers are most
popular and what could be learned from that. Like the original author, I also believe that most nixpkgs maintainers are
privacy-focused people who care deeply about FOSS.

But this belief didn't actually hold true for nixpkgs, or at least it didn't in 2024 when the original article was
written.

> I suppose the takeaway from this analysis is that developers prefer custom domain emails.
> For those not using custom domains, most still rely on Google.

I was curious to see if this belief now holds true in late 2026 and actually wanted to go **further** than the original
article. The author only parsed the email addresses available in the maintainers list, but most of them (54%) used
custom domains, so no conclusions could be drawn about their providers.  
But we could use a domain lookup utility like `dig` to actually see **which provider is behind a custom domain**.

I've had my own experience running an email server on my own infrastructure, and it didn't go *exactly* well. There are
multiple issues with modern email infrastructure that make hosting your own email server impractical or even unusable in
some cases. Thus, I expect that such experimentation is going to return a list of mostly well-known providers hiding
behind custom domains.

First, we have to obtain a list of all nixpkgs maintainers. This is easy because the list is
[publicly available](https://raw.githubusercontent.com/NixOS/nixpkgs/refs/heads/master/maintainers/maintainer-list.nix),
and a script to convert it to JSON already exists, *courtesy of the original article*.

```bash
curl https://raw.githubusercontent.com/NixOS/nixpkgs/refs/heads/master/maintainers/maintainer-list.nix \
 | nix eval --json -f - > maintainer-list.json
```

Next, in a Python script, I will filter out maintainers without a specified email address. I'm too lazy to fetch the
missing addresses using the GitHub API and the maintainers' GitHub IDs, but that's the path the aforementioned article
followed.

```python
import json

with open('maintainer-list.json') as file:
    data = json.load(file)

maintainers = [
    {'githubId': u['githubId'], 'email': u.get('email', None)}
    for u in data.values()
    if isinstance(u, dict) and u.get('githubId') is not None
]

maintainers_with_email = {r["githubId"]: r["email"] for r in maintainers if r["email"]}
missing = [r["githubId"] for r in maintainers if r["email"] is None]

print(
    f"{len(maintainers)} maintainers in total: {len(maintainers_with_email)} with email address provided, {len(missing)} without one")
```

This gives us an estimate of the size of our dataset:
`5378 maintainers in total: 4842 with email address provided, 536 without one`

We can compare the domain parts of the email addresses to estimate the popularity of each email domain. This approach is
not the most rigorous one, as the same entity can have multiple email domains like Proton, which has `proton.me`,
`protonmail.com`, and `pm.me`, but it should still be useful for getting a rough estimate.

```python
from collections import Counter

domains = Counter(
    email_addr.split("@")[-1].strip().lower()
    for email_addr in maintainers_with_email.values()
)

print("\n20 most common domains in email addresses:")
print('-' * 55)
print(f"{'provider':<39} {'emails':>7} {'share':>6}")
print('-' * 55)
for provider, count in domains.most_common(20):
    print(f"{provider:<39} {count:>7} {count / domains.total():>7.1%}")
```

Running that snippet gives us the awaited result:

```
20 most common domains in email addresses:
-------------------------------------------------------
provider                                 emails  share
-------------------------------------------------------
gmail.com                                  1569   32.4%
proton.me                                   138    2.9%
protonmail.com                              122    2.5%
pm.me                                        68    1.4%
outlook.com                                  56    1.2%
icloud.com                                   30    0.6%
posteo.de                                    25    0.5%
hotmail.com                                  25    0.5%
riseup.net                                   20    0.4%
tutanota.com                                 17    0.4%
mailbox.org                                  16    0.3%
disroot.org                                  16    0.3%
tuta.io                                      15    0.3%
posteo.net                                   15    0.3%
duck.com                                     13    0.3%
googlemail.com                               13    0.3%
fastmail.com                                 12    0.2%
gmx.de                                       11    0.2%
qq.com                                       11    0.2%
web.de                                        8    0.2%
```

**Gmail ranks first with 32.4% of all maintainers**, followed by Proton, whose three domains host emails for a little
under 7% of all nixpkgs maintainers. Other notable mentions include Outlook at around 1.2%, iCloud with 30 addresses
(0.6%), and DuckDuckGo's free email alias system with 13 addresses (0.3%).

Outlook ranks *higher* than I would have expected. I knew Outlook was widely used in the corporate sense, mostly under
company-managed custom domains, but it looks like outlook.com is still relevant for personal use.

Now comes the fun part and the main reason I'm writing about this experiment: our list of the 20 most common domains
only includes around 2,200 email addresses, **so where did the others go?**

It's no surprise that we developers tend to tinker a lot with our everyday tools, and that includes, of course, email.
That said, a very popular way to do that is with custom email domains: `me@dawoox.dev`, `git@patate.dev`, and the like.

We can use a utility like `dig` on Linux, or the Python library dnspython, to check the domain's DNS records to see
which server actually handles the email. My theory is that most of us aren't using custom servers but are
using a third-party provider (paid or not) to handle email for our domains.

Proton, for example, [provides this feature](https://proton.me/support/custom-domain) to all of its paid customers. As
for me, I'm using an [OVH service](https://www.ovhcloud.com/fr/emails/email-pro/) to do that for a couple of euros per
month.

```python
print("\nfetching MX record of domains, this step take a long time (~10mn)... ", end='')
for domain, count in domains.items():
    try:
        answer = dns.resolver.resolve(domain + '.', 'MX', raise_on_no_answer=False)
        if answer.rrset is None:
            key = '(no MX)'
        else:
            primary = sorted(answer.rrset, key=lambda r: r.preference)[0]
            key = base_domain(primary.exchange.to_text().rstrip('.').lower())
    except dns.resolver.NXDOMAIN:
        key = '(NXDOMAIN)'
    except dns.exception.DNSException:
        key = '(lookup failed)'

    provider_emails[key] += count
print("Done.")

print("\n20 most common email providers (by MX records):")
print('-' * 55)
print(f"{'provider':<39} {'emails':>7} {'share':>7}")
print('-' * 55)
for key, n in provider_emails.most_common(20):
    print(f"{key:<39} {n:>7} {n / provider_emails.total():>7.1%}")
```

A couple of minutes later *(sorry, I was too lazy to make this multithreaded or even asynchronous)*, we get more insight
into what's hiding behind those domain names.

```
20 most common email providers (by MX records):
-------------------------------------------------------
provider                                 emails   share
-------------------------------------------------------
google.com                                 1863   38.5%
protonmail.ch                               600   12.4%
messagingengine.com                         320    6.6%
outlook.com                                 149    3.1%
mailbox.org                                 116    2.4%
icloud.com                                  104    2.1%
cloudflare.net                               98    2.0%
migadu.com                                   96    2.0%
tutanota.de                                  76    1.6%
simplelogin.co                               47    1.0%
posteo.de                                    46    1.0%
(NXDOMAIN)                                   42    0.9%
(no MX)                                      38    0.8%
zoho.com                                     35    0.7%
purelymail.com                               31    0.6%
(lookup failed)                              24    0.5%
infomaniak.ch                                22    0.5%
mxrouting.net                                21    0.4%
riseup.net                                   20    0.4%
mailgun.org                                  18    0.4%
```

The results are kinda interesting.    
First up the list is obviously Google, but this time with 38.5% of all email addresses, so a non-negligible number of
users use Gmail behind their custom domains. This includes mostly professional users with
paid [Google Workspace accounts](https://knowledge.workspace.google.com/admin/gmail/activate-gmail-with-google-workspace-your-company)
for their company email.  
Right behind it is Proton at 12.4%: a 5.4-percentage-point increase. **Almost half of nixpkgs maintainers using Proton
are using custom domains.**  
At 6.6%, the domain `messagingengine.com` makes its first appearance. It's actually a domain owned and run
by [Fastmail](https://www.fastmail.com/), which accounted for only 0.2% before.  
iCloud also shows up with a higher share (2.1% now, compared with only 0.6% before). That makes sense, as iCloud offers
this [custom domain option](https://support.apple.com/en-us/102540) to its paid customers.

Other notable mentions include Cloudflare, Infomaniak, and `(no MX)`.

I was expecting Cloudflare to rank higher, as its email routing feature is available on the free plan and is a feature I
rely on heavily for several of my domains. It allows you to receive email on your custom domain and forward it, or not,
based on a ruleset, to one or more email addresses hosted elsewhere.

Infomaniak is a name I was also hoping to see. It's a Swiss-based cloud provider run by a foundation committed to
safeguarding its promise of "A 100% sovereign cloud, free from any non-European influence".

`(no MX)` is a funny one. I'm not going to shame anyone; everyone knows that if something is not working, **it's always
DNS**, and forgetting to set your MX records is one thing that can explain why.

## So what?

I find these overall results interesting because they show a bigger picture than the original article.  
Still, the self-reliant, sovereignty-minded part of me finds it pretty sad to see that nowadays most of our critical
services are centralized in the hands of a few entities with too much control over our lives.  
But the picture is not *too* dark. Proton is a good option, and it has a healthy share of users, as we saw.    
And even after our MX record snooping, the top 20 account for only 3,769 addresses, leaving **1,073 addresses**
remaining, those could belong to institutions, sure, but also to enthusiasts running their own mail servers, as the web
was [originally designed to be](https://medium.com/@timberners_lee/marking-the-webs-35th-birthday-an-open-letter-ebb410cc7d42):
all hope is not lost.

## Source code

[Available on GitHub](https://gist.github.com/Dawoox/f0672ffb16e7750c918f03d3d4656249)  
[Download here](nixpkgs-maintainers-demographics.tar.gz)
