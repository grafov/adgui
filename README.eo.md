# AdGUI por AdGuard VPN

**Languages:** [English](README.md) · [Русский](README.ru.md) · [Esperanto](README.eo.md)

Simpla grafika interfaco por regi la CLI de AdGuard VPN sur Linux-surtableto (XLibre/X11 aŭ Wayland).
AdGuard oficiala ofertas GUI-ojn por Mac, Android kaj Windows, sed por Linux mankas ĝi :(

> La projekto ne provizas VPN-funkcion. Ĉi tio estas nur helpa interfaco, kiu ĉirkaŭas
> la veran VPN-aplikaĵon (`adguardvpn-cli`) por pli komforta uzo en
> surtabla medio. La projekto ne havas rilaton al AdGuard nek al iu el iliaj
> produktoj.

La GUI proksime imitas la funkciojn de AdGuard VPN sur Linux. Bedaŭrinde,
la Linux-versio de AdGuard VPN havas malpli da funkcioj ol la samaj por
Mac/Windows.

Laboro daŭras, sed la aplikaĵo estas plene funkcia.

![informoj pri konekto](doc/scr2.png)

![regiona kontrolo](doc/scr1.png)

Mallonga [demonstrado en Youtube](https://www.youtube.com/shorts/UpstI1BD-rE).

## Lokalizado

La interfaca lingvo estas aŭtomate elektita laŭ la sistema loko (Fyne i18n). Subtenataj UI-lingvoj: la angla (`en`), la rusa (`ru`) kaj Esperanto (`eo`). La angla estas rezerva, kiam kongrua traduko ne disponeblas.
Aliaj lingvoj povas esti aldonitaj poste.

## Instalado

Nuntempe ne proviziĝas duumaj dosieroj. Sur via maŝino devus esti Go-programada medio.

Ĝi provas instali en la protektitan dosierujon `/usr/local/bin`, kio postulas root-privilegiojn. Uzu `sudo`, `doas` aŭ alian taŭgan komandon per la SUDO-medio:

`SUDO=sudo make install`

Aŭ uzu `PREFIX` por instali en alian dosierujon, ekzemple hejme:

`PREFIX=~/bin make install`

## Funkcioj

Unue ensalutu en vian AdGuard-konton per `adguardvpn-cli`. Mi ne enmetis ĉi tiun parton en la GUI por simpleco, ĉar vi bezonas ĝin nur unufoje.

### Subteno por taskopleto

En la menuo de la taskopleta piktogramo disponeblas: montri la panelon, konekti al loko, konekti al la antaŭa loko, agordi retejajn ekskluzivojn, malkonekti la VPN.

### Retejaj ekskluzivoj (langeto «Domajnoj»)

La langeto «Domajnoj» en la panelo permesas administri retejajn ekskluzivojn por AdGuard VPN. Vi povas agordi, kiuj domajnoj ĉirkaŭpas aŭ uzas la VPN-konekton.

#### Ekskluzivaj reĝimoj
- **Ĝenerala reĝimo**: domajnoj en la listo estas ekskluzivitaj de la VPN (trafiko iras rekte)
- **Selektiva reĝimo**: nur domajnoj en la listo uzas la VPN-konekton

#### Aŭtomata persistado
La ekskluzivaj listoj estas apartigitaj laŭ reĝimo (ĝenerala kaj selektiva) kaj aŭtomate konserviĝas en la jenaj lokaj dosieroj ĉe ĉiu ŝanĝo (aldono, algluo, importo, forigo, vakigo):
- Ĝenerala reĝimo: `~/.config/adgui/site-exclusions/general.txt`
- Selektiva reĝimo: `~/.config/adgui/site-exclusions/selective.txt`

Kiam vi ŝanĝas ekskluzivan reĝimon, la aktiva listo estas konservita en la respondan dosieron, kaj la listo de la nova reĝimo estas aŭtomate ŝargita kaj aplikita al la CLI.

#### Administrado de domajnoj

- **Filtri/aldoni**: uzu la tekstkampon supre por filtri ekzistantajn domajnojn aŭ enigi novan domajnan nomon
- **Aldoni**: alklaku la butonon «Aldoni» por aldoni la domajnon el la tekstkampo al la ekskluziva listo
- **Forigi**: alklaku la butonon «X» apud iu domajno por forigi ĝin el la listo

### Importo/Eksporto

La butonoj «Importi» kaj «Eksporti» permesas konservi kaj restarigi domajnajn ekskluzivajn listojn por la **nuna ekskluziva reĝimo** (ĝenerala aŭ selektiva).

#### Eksporto

Alklaku **Eksporti** por konservi domajnojn el la nuna reĝimo en dosieron:

1. Malfermiĝas sistema konservdialogo kun la defaŭlta dosiernomo `<mode>.adgui` (`general.adgui` aŭ `selective.adgui`)
2. Elektu la celan vojon kaj dosiernomon (defaŭlta etendo: `.adgui`)
3. La dosiero estas konservita kun la eksportitaj domajnoj

**Noto**: nur la nuntempe filtritaj/videblaj domajnoj estas eksportitaj. Se filtro estas aktiva, nur kongruantaj domajnoj estos konservitaj. Malplenigu la filtron por eksporti ĉiujn domajnojn en la nuna reĝimo.

#### Importo

Alklaku **Importi** por ŝargi domajnojn el dosiero en la **nunan ekskluzivan reĝimon**:

1. Malfermiĝas sistema malfermdialogo (iu dosieretendo)
2. Elektu dosieron kun unu domajno po linio
3. Novaj domajnoj estas aldonitaj al la listo de la nuna reĝimo en adgui kaj tuj aplikitaj al AdGuard VPN
4. Duobligitaj domajnoj (jam en la listo) estas aŭtomate preterlasitaj

La importo montras progresindikilon kaj aktualigas la liston post fino. Importitaj domajnoj estas persistitaj en la dosieron de la nuna reĝimo (`general.txt` aŭ `selective.txt`).

### Migrado

Se vi havas malnovajn unuigitajn tekstajn ekskluzivajn dosierojn, vi povas migrigi ilin al la novaj reĝimspecifaj dosieroj per la provizita Python-skripto:

```bash
python3 scripts/migrate-site-exclusions.py --target-mode [general|selective]
```

Defaŭlte la skripto skanas la malnovan dosierujon `~/.local/share/adgui/site-exclusions/` kaj kunfandas ĉiujn dosierojn (krom `general.txt` kaj `selective.txt`) en la celan reĝiman dosieron ĉe `~/.config/adgui/site-exclusions/` kun aŭtomata deduplikado. Vi ankaŭ povas eksplicite indiki enirajn dosierojn per la flago `--input <path>`.

#### Dosierformato

Eksportaj/importaj dosieroj estas simpla teksto kun unu domajno po linio. La defaŭlta eksporta etendo estas `.adgui`. Vi ankaŭ povas permane krei aŭ redakti ĉi tiujn dosierojn:

```
example.com
subdomain.example.org
another-site.net
```

### Lokaj legosignoj («Konekti al...»)

La lokselektilo **Konekti al...** permesas legosigni VPN-lokojn per la stelkolumno dekstre. Legosignitaj lokoj estas konservitaj en:

- `~/.config/adgui/bookmarks`

Alklaku la kolumnan kaplinion **★** por baskuli ordigon kun legosignitaj lokoj unue. Alklaku la stelon en vico por aldoni aŭ forigi legosignon sen konekti.

Landaj flagoj en la loklisto uzas SVG-aktivaĵojn el [lipis/flag-icons](https://github.com/lipis/flag-icons) (permesilo MIT), enigitajn en la aplikaĵan duumon.

### IP-Regiono (landa detekto)

Ĉi tiu funkcio estas implementita nur en AdGUI kaj ne estas parto de AdGuard VPN. Ĝi povas helpi analizi la efikecon de via VPN-konekto. La langeto **IP-Regiono** en la panelo kontrolas, kiel GeoIP-datumbazoj kaj popularaj retaj servoj klasifikas vian nunan elirantan IP-adreson. Uzu ĝin por kontroli, ĉu AdGuard VPN direktas trafikon tra la atendata lando, aŭ por vidi, ĉu malsamaj servoj malkonsentas pri via loko.

La kontrolo funkcias nur kiam vi premis **Kie mi estas?** — malfermi la langeton ne kontaktas la reton. Dum skanado progresstrio montras la nunan servon kaj provokalkulilon; premu la butonon denove por nuligi.

#### Rezultoj

- **Resumo** — plej oftaj landoj laŭ kiom da servoj raportis ĉiun ISO-kodon (IPv4- kaj IPv6-procentoj kiam ambaŭ disponeblas)
- **Servotabelo** — landkodoj po servo por IPv4 kaj IPv6
- **VPN-komparo** — kiam konektita, komparas la elektitan VPN-lokon kun la konsento de eksteraj kontroloj; malkongruoj estas markitaj per `!` en la tabelo

Ĉefaj proboj demandas GeoIP-API-ojn (MaxMind, ipinfo.io, Cloudflare, ip-api.com kaj aliaj). Propraj proboj inferas regionon el respondoj de popularaj retejoj (Google, YouTube, Netflix, Spotify, Steam kaj aliaj). La logiko estas portita el [Davoyan/ipregion](https://github.com/Davoyan/ipregion) (permesilo MIT).

#### Nedevigaj API-ŝlosiloj

Iuj servoj por regiona kontrolo akceptas viajn proprajn API-ŝlosilojn. Kreu `~/.config/adgui/service-keys` (formato INI) kun iu el:

```ini
IPREGISTRY_KEY=your_key
GEOAPIFY_KEY=your_key
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_API_KEY=your_api_key
AIRPORT_CODES_AUTH=your_token
```

Se la dosiero aŭ ŝlosilo mankas, enkonstruitaj demonstraj defaŭltoj estas uzataj kie disponeblas; aliaj servoj funkcias sen ŝlosiloj.

## Kodo per AI

Mi aktive uzas LLM-ojn por generi grandajn partojn de kodo, testoj, kodrecenzo, lokalizado kaj dokumentado por ĉi tiu projekto. Ĝi estis eksperimento krei GUI (per la Fyne-kadro) en Go per LLM-oj. Ĝenerale sukcesa, kvankam mi devis permane korekti iujn partojn. La tuta kodo estas recenzita de mi.

## Starware

Ĉi tiu programaro estas starware :) Se vi trovas la kodon utila, ne forgesu voĉdoni por ĉi tiu deponejo per stelo ⭐!

## Permesilo

Laŭ la kondiĉoj de GPL v3.
