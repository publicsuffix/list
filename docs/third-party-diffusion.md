### Third Party Diffusion

The Public Suffix List is incorporated for a number of uses, and the project does not prescribe how it is or is not used.

One use case that has evolved which places increasing burden on the volunteer PSL maintainers is where some providers point to the PSL as being a place to get included in order to add or change functionality within their systems.

This third party diffusion includes, in some cases, where the presence or absence of entries is leveraged by companies to point customer requests away to an external party in order to keep their own support or customer flow simplified.

## A special note to operating systems, social media giants, and other companies, projects or vendors who introduce rate limits and refer people to resolve their issue via the PSL:

# Don't.

Instead, address your customers directly. While referring folks away may make for a simple checkbox next to done for your project or elegant customer service FAQ flow for you or your company, it dumps irritated folks who are affected onto the volunteers that help maintain this project.

It should go without mention that it is inappropriate to do so, but mention it we must. The volunteer resourcing drain is annoying and disrespectful, but it is secondary to directing your customers/clients to potentially request things that might break their websites if not handed off after careful qualification BEFORE handing them off. It is also a crucially important fact about the private section of the PSL that it infers ZERO security.

### The #PRIVATE section comes from the public (irony noted). There should _zero trust_ or security assumptions made about these entries.

The inclusion of a domain name that is core to a project may have unexpected affectations that were not desired, and should be reviewed with the support department of the referring party to ensure that they have walked you through the impacts and consequences of PSL inclusion so that you have an intentional experience / outcome with the systems/services that referred you, and have made every attempt to resolve it with them.

In every case, the most appropriate solution to issues such as those that point towards having entries in the PSL should first be resolved with the referring party, as utilizing the PSL as a workaround for engineered limits likely sidesteps important security or other intentional concerns that fed the logic to have those limits.

* For issues with Let's Encrypt, contact Let's Encrypt.
* For issues with Cloudflare, contact Cloudflare.
* For issues with Adsense Subdomains, contact [Google Adsense Support](https://support.google.com/adsense/gethelp)
* For issues with IOS, contact Apple.
* For issues with GitLab, contact GitLab.
* For issues with Facebook Pixel, contact Facebook. They now have a review process for requests for addition. During that review the following things will are checked:
    - The domain(s) requested for addition have subdomains in large numbers
    - The domain(s) requested for addition do indeed appear to represent separate business entities which require cookie separation
    - The domain(s) requested for addition are owned by the requestor and will continue to be owned for at least two years
    - The domain(s) requested for addition are not running functionality which would break with a PSL addition (e.g. login/registration, cookie functionality, JavaScript and dynamic behaviors)
