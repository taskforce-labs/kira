# Buyer Receives and Completes a Checklist

## Job to be done

- **As a** buyer navigating a complex process (e.g. buying a home)
- **I want** a tailored, interactive checklist that tells me **which stages of the process to complete and in what order**
- **So that** I feel **organised**, **confident about what comes next**, and **never miss a critical action**

### Supporting evidence

**Perceived Problem:** Buyers juggle email, PDFs, and ad hoc advice; actions delayed (finance, legal, inspections) and can be disrupting to transactions.
**Signals:** Anecdotal feedback from my brother in-law who is a realtor in the industry for 15 years
**Assumptions:** A branded, ordered checklist increases completion and perceived confidence more than a static document; multi-track mode is needed only for a defined slice of buyers.

### Measures of success

In practice, **refine this block last**—after the **User Flow** stages and scenario variants below are stable—so indicators match the journey you actually documented. It stays here so “why” and “how we’ll know” stay together for readers.

- **Leading indicators:** Link open rate, **checklist viewed** (anonymous), **registration** after view, **items completed per week**, **return sessions**, **professional enquiries** from checklist **items**, **e-sign** completion where applicable.
- **Lag / business:** Time-to-key milestones (where measurable), realtor-reported readiness, NPS or CSAT on “clarity of what comes next” (if sampled).
- **Falsify *So that*:** Flat or declining completion despite traffic; high abandonment right after register; realtors see **no** useful engagement signals.
- **Review cadence:** Compare this doc to analytics and interviews on an agreed schedule (e.g. first 30 days after launch, then quarterly); update **Supporting evidence** when new facts land.

## User Flow

### Background


### Stage 1: Buyer receives the checklist link

The buyer receives a link to their personalised checklist via email, SMS, or directly from their realtor. The checklist may have been sent by a realtor or shared by another buyer (see *Buyer Shares a Checklist* for the sharer’s flow). The message comes from the realtor's organisation and includes a brief description of what the checklist contains (e.g., "Here's your personalised Home Buying Guide from Acme Realty").

### Stage 2: Buyer opens the checklist link

The buyer taps or clicks the link and is taken to the checklist view in the browser (or mobile app if installed). The link may be the generic checklist (`site.com/org/checklist-name`) or a personal share (`site.com/org/checklist-name/ABC123`); see *URL structure and claim state* below. The page renders with the sending realtor's organisation branding (logo, colours, theme). No login is required to view the checklist initially — the share ID (or generic checklist) grants read access.

### Stage 3: Buyer views the checklist overview

The buyer sees the full checklist with:

- A clear title (e.g., "Sarah's Home Buying Guide")
- A list of **items**, grouped into logical sections (e.g., "Finance", "Legal", "Property Search", "Closing")
- Each item has a short description, and some may include tips, links, **documents to sign**, or recommended professionals
- A progress indicator showing overall completion (e.g., "0 of 14 items completed")
- Optionally, when the checklist supports multiple tracks (see *Checklist-level and track-level items* below), a way to add or switch between tracks (e.g. properties, cars) so that track-level **items** (sub-tasks per track) are shown and tracked per track

### Stage 4: Buyer registers or logs in (if not already authenticated)

To mark items as complete and save progress, the buyer needs to register. A prompt appears encouraging registration with name, email, and phone number. If arriving via a direct realtor send, the buyer's email may already be in the system — they just confirm and set up access. Registration goes through WorkOS AuthKit.

### Stage 5: Buyer begins completing checklist items

The buyer works through items at their own pace, checking them off as they complete them:

- "Research areas you want to live in" — checked
- "Get mortgage agreement in principle" — checked
- "Find a solicitor" — the buyer taps this item and sees a recommended solicitor with an option to make an enquiry

Each completion updates the progress indicator and generates an engagement signal visible to the realtor.

### Stage 6: Buyer tracks progress over time

The buyer returns to the checklist over days or weeks, picking up where they left off. The checklist remembers their progress. They can see which items are done, which are next, and how far through the overall process they are. The progress state syncs in real time across devices.

### Stage 7: Buyer connects with recommended professionals

At relevant checklist **items**, the checklist surfaces recommended third-party professionals (e.g., a solicitor or mortgage broker). The buyer can view the professional's details and submit an enquiry directly from the checklist. The enquiry includes the buyer's registered details and the referrer's information (the realtor who sent the checklist).

---



## URL structure and claim state

- **Generic checklist**: `site.com/org/checklist-name`. Always shows the checklist template with no progress. Used when no specific share is in play (e.g. a generic link from the realtor, or when a claimed personal link is viewed by someone who is not logged in).

- **Personal share**: `site.com/org/checklist-name/ABC123`. The segment `ABC123` is a unique share ID linked to one recipient. The share is **unclaimed** until that recipient registers or logs in and claims it; until then, the link shows the full checklist (read-only or with a register CTA). Once **claimed**, the share is tied to that buyer's (user's) account and progress is stored against it.

- **Claimed share, viewer not logged in**: If someone opens a personal link that is already claimed (e.g. a forwarded email or a link meant for another buyer (user)) and they are not logged in — or they are logged in as a different buyer (user) — the system does not show the claimer's progress. It shows the **generic checklist** (no progress) and a message such as "Log in to view your progress" or "Get started". The link remains useful (they can see the checklist and choose to register or log in) without exposing another buyer's (user's) data.

- **Claiming from a link meant for someone else**: If that visitor registers or logs in from that page and they are not the original claimer, the system does not assign them to the existing share ID. It **generates a new share ID** for them and redirects to the new URL (e.g. `site.com/org/checklist-name/XYZ789`). The realtor is notified that a new share was claimed (e.g. from a forwarded or repurposed link), so they have visibility into who actually claimed and from where.

## Checklist-level and track-level items (multiple “things” per checklist)

A checklist can be structured so that the buyer (user) is pursuing more than one **track** — i.e. more than one of the “thing” the checklist is about. In home buying a track is a property; in car buying it could be a car; in other domains it might be a job application, a university, etc. Each track has its own sub-tasks. The checklist template distinguishes between:

- **Checklist-level items** — completed once for the whole checklist. Examples (home buying): "Get finance pre-approval", "Find a solicitor", "Research areas you want to live in". Examples (car buying): "Set your budget", "Check insurance costs". When the buyer (user) completes them, they are marked done for the whole checklist.

- **Track-level items** — completed per track; each track has its own instances of these items (sub-tasks). Examples (home buying): "Make an offer", "Register for auction", "Arrange a property survey", "Exchange contracts". Examples (car buying): "Test drive", "Get a quote", "Arrange finance for this vehicle". The buyer (user) adds tracks (e.g. properties, cars) and completes track-level items separately for each.

This pattern keeps the system reusable: the same architecture (checklist + optional tracks + checklist-level vs track-level items) applies across use cases. The current product focus is home buying, so “track” is often a property; the doc and architecture can use the generic terms so future use cases (e.g. car buying, applications) fit without new concepts.

**UI simplicity: zero or one track.** When no track has been added, or when only one track exists, the checklist should appear as a **simple linear list** — all items (checklist-level and track-level) look like primary-level rows. No tabs, no track selector, no “per track” grouping. Only when a **second** track is added does the UI switch to track-aware mode: the buyer (user) sees a way to switch between tracks (e.g. dropdown or tabs) and track-level items appear as sub-tasks under the selected track. So the default experience is a straight list; the extra structure appears only when it’s needed.

**How it works for the buyer (home buying):** The buyer has one checklist (e.g. "Home Buying Guide") tied to their share. They can **add tracks** (e.g. properties: "Flat on Oak Street", "Auction – Lot 12") from within the checklist. With zero or one track, the checklist looks like a single linear list. With two or more tracks, checklist-level items still appear once; track-level items appear in a track-specific context: the buyer selects a track (e.g. via a dropdown or tabs) and sees that track’s sub-tasks, with progress stored per track. The progress indicator can reflect overall progress (e.g. "8 of 14 checklist-level items done; Property A: 2 of 4 track items; Property B: 1 of 4 track items"). Enquiries can optionally be associated with a track so the professional knows which purchase (or car, etc.) the enquiry relates to.

## Checklist items that require a signature

Some checklist **items** can be **"Sign [document]"** — for example "Sign your offer letter", "Sign mortgage agreement", or "Sign engagement letter with solicitor". The realtor (when building the checklist) or the professional (when adding follow-up tasks) attaches a document to the item or task and marks it as requiring signature. The buyer completes the item by signing the document. **E-signature** can be delivered first via an **integration** (e.g. DocuSign): the buyer is sent to or shown the third-party signing flow; on completion, the item is marked done on the checklist and the realtor/professional sees confirmation. The platform may later offer a **native e-signature** capability to replace or complement the integration (see *Future Product Directions* in PRODUCT.md). Signing stays on-checklist so progress, reminders, and referrer context remain in one place.

## Scenarios

### Scenario 1: First-time home buyer receiving their first checklist
Sarah has never bought a home before. Her realtor sends her a "First-Time Buyer's Guide" after their initial meeting. Sarah opens the email on her phone, taps the link, and sees a clear, branded checklist. She registers with her details and starts checking off items she has already done (e.g., "Decide on your budget"). The structured format immediately reduces her anxiety about the process.

### Scenario 2: Buyer returning to continue progress
James received his checklist two weeks ago and completed the first three items. He returns today on his laptop by clicking the link in his original email (or by logging in directly). The checklist shows his previous progress — three items checked off with green ticks. He continues from where he left off, completing "Arrange a property survey". The progress bar updates from 3/12 to 4/12.

### Scenario 3: Buyer completing an item that connects them with a service provider
Emily reaches the "Instruct a solicitor" item on her checklist. The item includes a recommendation: "Smith & Partners — recommended by your realtor". Emily taps "Make an enquiry", which opens a pre-filled form with her details. She submits the enquiry. The solicitor receives the enquiry with Emily's contact information and a note that she was referred via her realtor at Acme Realty.

### Scenario 4: Buyer on mobile vs desktop
Tom starts his checklist on his phone during his commute, checking off a couple of items. Later that evening, he opens the same link on his laptop to review the detailed notes on the "Arrange a mortgage" item. His progress is synced — the items he checked on mobile appear as completed on desktop. The responsive layout adapts to each screen size.

### Scenario 5: Buyer with multiple lists from the same realtor
A buyer has received two lists from their realtor — an initial "Getting Started" guide and a later "Making an Offer" guide. When the buyer logs in, they can see both lists and their respective progress. They can switch between lists depending on which part of the buying process they are currently focused on.

### Scenario 6: Buyer views the checklist but does not register
A buyer opens the checklist link and browses the checklist items without registering. They can see the full checklist and read descriptions, but items are not interactive (cannot be checked off) until they register. A gentle prompt at the bottom encourages registration: "Register to save your progress and get personalised recommendations." The realtor still receives a "checklist viewed" signal even without registration.

### Scenario 7: Someone opens a link that is already claimed (e.g. forwarded link)
A realtor sent a checklist to a buyer who registered and claimed their share (e.g. `site.com/org/checklist-name/ABC123`). Later, someone else opens the same link — the buyer forwarded the email to a family member, or a friend found the link in a shared chat. Because the share is claimed and the visitor is not logged in (or is logged in as a different buyer (user)), they do not see the original buyer's progress. They see the **generic checklist** with no progress and a message: "Log in to view your progress" or "Get started". If they then register or log in, the system does not attach them to the existing share; it **generates a new share ID** (e.g. `site.com/org/checklist-name/XYZ789`) and redirects them there. The realtor is notified that a new share was claimed from a link that was already claimed (e.g. forwarded), so they know who actually claimed and that the link was reused.

### Scenario 8: Generic link vs personal share — unique URL per recipient
A realtor shares the same checklist with multiple buyers. The **generic** checklist URL is `site.com/org/checklist-name` (e.g. in a signature or on a leaflet). When someone opens that generic link, the system can issue a new **personal** share and redirect them to `site.com/org/checklist-name/ABC123`, so each recipient ends up with their own unique URL. Personal shares are **unclaimed** until the recipient registers or logs in; once claimed, progress is tied to that share ID. So: (1) generic link always shows the checklist template; (2) personal links are unclaimed until the buyer (user) claims them; (3) claimed personal links show progress only to the logged-in claimer — everyone else sees the generic checklist and a prompt to log in or get started; (4) if someone claims from a link meant for another buyer (user), they get a new share ID and the realtor is notified. This keeps progress private and gives the realtor clear attribution per share.

### Scenario 9: Buyer with multiple tracks (properties) on the same checklist
David is bidding at an auction and also considering a private sale. He has one checklist from his realtor. He adds two tracks (properties): "Auction – Lot 12" and "42 Maple Drive (private sale)". Checklist-level items like "Get mortgage agreement in principle" and "Find a solicitor" appear once; he completed them earlier, so they show as done for the whole checklist. Track-level items (sub-tasks) are shown per track: under "Auction – Lot 12" he sees "Register for auction" (done), "Submit bid" (not done), "Pay deposit if successful" (not done); under "42 Maple Drive" he sees "Make an offer" (done), "Arrange survey" (in progress). He can switch between tracks (e.g. tabs or a selector) to focus on the next priority **item** for each track. When he submits an enquiry to the recommended solicitor, he can optionally tag it with a track so the solicitor knows which purchase it relates to. The realtor sees David's overall progress plus per-track completion. (The same pattern would apply for a car buyer with multiple cars: list-level items once, track-level sub-tasks per car.)

### Scenario 10: Buyer completes a "Sign document" item (e-signature)
The checklist includes an item "Sign your offer letter" with a document attached by the realtor. When the buyer taps the item, they are taken through an e-signature flow (e.g. via a DocuSign integration or a future native signing experience). They sign the document; on completion, the item is marked done on the checklist and the realtor sees that the offer has been signed. The buyer's progress updates without leaving the checklist context. See *Checklist items that require a signature* above and PRODUCT.md (Future Product Directions — document signing).