package legal

const privacyPolicyTemplate = `# Privacy Policy

**Effective date:** {{EFFECTIVE_DATE}}  
**Version:** 1.0

{{COMPANY_NAME}} ("**we**", "**us**", or "**our**") operates **{{APP_NAME}}** (the "**Service**"), including our mobile applications and organizer dashboard. This Privacy Policy explains how we collect, use, disclose, and protect personal information when you use the Service.

By creating an account or using the Service, you agree to this Privacy Policy. If you do not agree, do not use the Service.

## 1. Who we are

The Service is developed and provided by **{{COMPANY_NAME}}**.

- **Contact:** [{{CONTACT_EMAIL}}](mailto:{{CONTACT_EMAIL}})
- **Other products:** [{{PRODUCTS_URL}}]({{PRODUCTS_URL}})

## 2. Information we collect

### 2.1 Information you provide

- **Account details:** name, email address, phone number (optional), password or authentication via third-party sign-in (e.g. Google).
- **Profile and preferences:** notification settings, timezone, and similar in-app choices.
- **Organizer content:** event titles, geofence definitions, consent templates, attendance notes, and related business data when you use organizer features.

### 2.2 Information collected automatically

- **Location data:** precise or approximate location when you grant permission, including for geofence entry/exit detection, clock-in validation, and attendance verification. Background location may be used when enabled for geofence monitoring.
- **Device and usage data:** device type, operating system, app version, IP address, timestamps, and diagnostic logs needed to operate and secure the Service.
- **Attendance and activity data:** clock-in/clock-out times, geofence events, QR scan metadata (when scan-to-clock-in is enabled), and activity history associated with events you join.
- **Push notification tokens:** device tokens used to deliver push notifications you opt into.

### 2.3 Camera and QR codes

If you use scan-to-clock-in or similar features, the app may access your device camera solely to read QR codes. We do not store photos or video from your camera unless explicitly stated for a specific feature.

### 2.4 Information from third parties

- **Authentication providers:** if you sign in with Google or similar services, we receive profile information permitted by your provider settings (such as name and email).
- **Event organizers:** organizers of events you join may receive attendance and location-verification status as part of event operations.

## 3. How we use information

We use personal information to:

- Provide, maintain, and improve the Service;
- Authenticate users and enforce account security;
- Process geofence-based attendance, notifications, and organizer reporting;
- Send transactional messages (e.g. verification codes, password reset, attendance alerts) and, with your consent, email or push notifications;
- Comply with law, prevent fraud, and protect the rights and safety of users and {{COMPANY_NAME}};
- Analyze aggregated, de-identified usage to improve reliability and features.

We do not sell your personal information.

## 4. Legal bases (EEA/UK users)

Where applicable, we process personal data on the basis of: performance of a contract (providing the Service), legitimate interests (security, product improvement, fraud prevention), consent (e.g. optional marketing or certain notifications), and legal obligation.

## 5. How we share information

We may share information with:

- **Service providers** who assist with hosting, email delivery, push notifications, analytics, and support, under contractual confidentiality and data-processing terms;
- **Event organizers** for events you join, limited to attendance and related operational data;
- **Authorities** when required by law or to protect rights, safety, and security;
- **Business transfers** in connection with a merger, acquisition, or asset sale, subject to this Policy.

We require processors to handle data only on our instructions and with appropriate safeguards.

## 6. International transfers

Your information may be processed in countries other than your own. Where required, we use appropriate safeguards (such as standard contractual clauses) for cross-border transfers.

## 7. Data retention

We retain personal information for as long as your account is active or as needed to provide the Service, resolve disputes, enforce agreements, and meet legal obligations. Location and attendance records may be retained according to organizer settings and applicable law. You may request deletion as described below.

## 8. Security

We implement administrative, technical, and organizational measures designed to protect personal information. No method of transmission or storage is completely secure; we cannot guarantee absolute security.

## 9. Your rights and choices

Depending on your location, you may have the right to:

- Access, correct, or delete your personal information;
- Object to or restrict certain processing;
- Withdraw consent where processing is consent-based;
- Port your data in a structured, commonly used format;
- Lodge a complaint with a supervisory authority.

To exercise these rights, contact us at **{{CONTACT_EMAIL}}**. We may verify your identity before responding.

You can control location, camera, and notification permissions in your device settings. Disabling location may limit geofence and attendance features.

## 10. Children's privacy

The Service is not directed to children under 13 (or the minimum age required in your jurisdiction). We do not knowingly collect personal information from children. Contact us if you believe we have collected such information and we will delete it.

## 11. Third-party links and services

The Service may link to third-party sites (including [{{PRODUCTS_URL}}]({{PRODUCTS_URL}})). Their privacy practices are governed by their own policies, not this one.

## 12. Changes to this Policy

We may update this Privacy Policy from time to time. We will post the updated version with a new effective date. Material changes may be communicated through the Service or by email where appropriate. Continued use after changes constitutes acceptance where permitted by law.

## 13. Contact us

Questions about this Privacy Policy or our data practices:

**{{COMPANY_NAME}}**  
Email: [{{CONTACT_EMAIL}}](mailto:{{CONTACT_EMAIL}})  
Products: [{{PRODUCTS_URL}}]({{PRODUCTS_URL}})
`
