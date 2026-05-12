package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/eralme/server/internal/articles"
	"github.com/eralme/server/internal/certificates"
	"github.com/eralme/server/internal/contact"
	"github.com/eralme/server/internal/games"
	"github.com/eralme/server/internal/intro"
	"github.com/eralme/server/internal/links"
	"github.com/eralme/server/internal/music"
	"github.com/eralme/server/internal/products"
	"github.com/eralme/server/internal/projects"
	"github.com/eralme/server/internal/series"
	"github.com/eralme/server/internal/skills"
	"github.com/eralme/server/internal/tools"
	"github.com/eralme/server/internal/workexperience"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const maxSampleRecords = 8

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect postgres: %v", err)
	}

	projectStore, err := projects.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate projects postgres: %v", err)
	}
	certificateStore, err := certificates.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate certificates postgres: %v", err)
	}
	articleStore, err := articles.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate articles postgres: %v", err)
	}
	workExperienceStore, err := workexperience.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate work experiences postgres: %v", err)
	}
	skillStore, err := skills.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate skills postgres: %v", err)
	}
	linkStore, err := links.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate links postgres: %v", err)
	}
	toolStore, err := tools.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate tools postgres: %v", err)
	}
	musicStore, err := music.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate music postgres: %v", err)
	}
	seriesStore, err := series.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate series postgres: %v", err)
	}
	gameStore, err := games.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate games postgres: %v", err)
	}
	productStore, err := products.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate products postgres: %v", err)
	}
	introStore, err := intro.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate intro postgres: %v", err)
	}
	contactStore, err := contact.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate contact postgres: %v", err)
	}

	projectSamples := sampleProjects()
	for _, idOrSlug := range sampleProjectIDsForCleanup() {
		if err := deleteProjectIfExists(ctx, projectStore, idOrSlug); err != nil {
			log.Fatalf("delete sample %q: %v", idOrSlug, err)
		}
	}
	for _, project := range projectSamples {
		if err := deleteProjectIfExists(ctx, projectStore, project.ID); err != nil {
			log.Fatalf("delete sample %q: %v", project.ID, err)
		}
		if err := deleteProjectIfExists(ctx, projectStore, project.Slug); err != nil {
			log.Fatalf("delete sample %q: %v", project.Slug, err)
		}
		if _, err := projectStore.Create(ctx, project); err != nil {
			log.Fatalf("create sample %q: %v", project.Slug, err)
		}
	}

	certificateSamples := sampleCertificates()
	for _, idOrSlug := range sampleCertificateIDsForCleanup() {
		if err := deleteCertificateIfExists(ctx, certificateStore, idOrSlug); err != nil {
			log.Fatalf("delete certificate sample %q: %v", idOrSlug, err)
		}
	}
	for _, certificate := range certificateSamples {
		if err := deleteCertificateIfExists(ctx, certificateStore, certificate.ID); err != nil {
			log.Fatalf("delete certificate sample %q: %v", certificate.ID, err)
		}
		if err := deleteCertificateIfExists(ctx, certificateStore, certificate.Slug); err != nil {
			log.Fatalf("delete certificate sample %q: %v", certificate.Slug, err)
		}
		if _, err := certificateStore.Create(ctx, certificate); err != nil {
			log.Fatalf("create certificate sample %q: %v", certificate.Slug, err)
		}
	}

	articleSamples := sampleArticles()
	validateSampleLimit("articles", len(articleSamples))
	for _, article := range articleSamples {
		if err := deleteArticleIfExists(ctx, articleStore, article.ID); err != nil {
			log.Fatalf("delete article sample %q: %v", article.ID, err)
		}
		if _, err := articleStore.Create(ctx, article); err != nil {
			log.Fatalf("create article sample %q: %v", article.ID, err)
		}
	}

	workExperienceSamples := sampleWorkExperiences()
	for _, idOrSlug := range sampleWorkExperienceIDsForCleanup() {
		if err := deleteWorkExperienceIfExists(ctx, workExperienceStore, idOrSlug); err != nil {
			log.Fatalf("delete work experience sample %q: %v", idOrSlug, err)
		}
	}
	for _, workExperience := range workExperienceSamples {
		if err := deleteWorkExperienceIfExists(ctx, workExperienceStore, workExperience.ID); err != nil {
			log.Fatalf("delete work experience sample %q: %v", workExperience.ID, err)
		}
		if err := deleteWorkExperienceIfExists(ctx, workExperienceStore, workExperience.Slug); err != nil {
			log.Fatalf("delete work experience sample %q: %v", workExperience.Slug, err)
		}
		if _, err := workExperienceStore.Create(ctx, workExperience); err != nil {
			log.Fatalf("create work experience sample %q: %v", workExperience.Slug, err)
		}
	}

	skillSamples := sampleSkills()
	for _, id := range sampleSkillIDsForCleanup() {
		if err := deleteSkillIfExists(ctx, skillStore, id); err != nil {
			log.Fatalf("delete skill sample %q: %v", id, err)
		}
	}

	skillCategorySamples := sampleSkillCategories()
	for _, idOrSlug := range sampleSkillCategoryIDsForCleanup() {
		if err := deleteSkillsInCategoryIfExists(ctx, skillStore, idOrSlug); err != nil {
			log.Fatalf("delete skills in category sample %q: %v", idOrSlug, err)
		}
	}
	for _, idOrSlug := range sampleSkillCategoryIDsForCleanup() {
		if err := deleteSkillCategoryIfExists(ctx, skillStore, idOrSlug); err != nil {
			log.Fatalf("delete skill category sample %q: %v", idOrSlug, err)
		}
	}
	for _, category := range skillCategorySamples {
		if err := deleteSkillCategoryIfExists(ctx, skillStore, category.ID); err != nil {
			log.Fatalf("delete skill category sample %q: %v", category.ID, err)
		}
		if err := deleteSkillCategoryIfExists(ctx, skillStore, category.Slug); err != nil {
			log.Fatalf("delete skill category sample %q: %v", category.Slug, err)
		}
		if _, err := skillStore.CreateCategory(ctx, category); err != nil {
			log.Fatalf("create skill category sample %q: %v", category.Slug, err)
		}
	}
	for _, skill := range skillSamples {
		if _, err := skillStore.CreateSkill(ctx, skill); err != nil {
			log.Fatalf("create skill sample %q: %v", skill.ID, err)
		}
	}

	linkSamples := sampleLinks()
	validateSampleLimit("links", len(linkSamples))
	for _, id := range sampleLinkIDsForCleanup() {
		if err := deleteLinkIfExists(ctx, linkStore, id); err != nil {
			log.Fatalf("delete link sample %q: %v", id, err)
		}
	}
	for _, link := range linkSamples {
		if _, err := linkStore.Create(ctx, link); err != nil {
			log.Fatalf("create link sample %q: %v", link.ID, err)
		}
	}

	toolSamples := sampleTools()
	validateSampleLimit("tools", len(toolSamples))
	for _, id := range sampleToolIDsForCleanup() {
		if err := deleteToolIfExists(ctx, toolStore, id); err != nil {
			log.Fatalf("delete tool sample %q: %v", id, err)
		}
	}
	for _, tool := range toolSamples {
		if _, err := toolStore.Create(ctx, tool); err != nil {
			log.Fatalf("create tool sample %q: %v", tool.ID, err)
		}
	}

	musicSamples := sampleMusic()
	validateSampleLimit("music", len(musicSamples))
	for _, entry := range musicSamples {
		if err := deleteMusicIfExists(ctx, musicStore, entry.ID); err != nil {
			log.Fatalf("delete music sample %q: %v", entry.ID, err)
		}
		if _, err := musicStore.Create(ctx, entry); err != nil {
			log.Fatalf("create music sample %q: %v", entry.ID, err)
		}
	}

	seriesSamples := sampleSeries()
	validateSampleLimit("series", len(seriesSamples))
	for _, entry := range seriesSamples {
		if err := deleteSeriesIfExists(ctx, seriesStore, entry.ID); err != nil {
			log.Fatalf("delete series sample %q: %v", entry.ID, err)
		}
		if _, err := seriesStore.Create(ctx, entry); err != nil {
			log.Fatalf("create series sample %q: %v", entry.ID, err)
		}
	}

	gameSamples := sampleGames()
	validateSampleLimit("games", len(gameSamples))
	for _, entry := range gameSamples {
		if err := deleteGameIfExists(ctx, gameStore, entry.ID); err != nil {
			log.Fatalf("delete game sample %q: %v", entry.ID, err)
		}
		if _, err := gameStore.Create(ctx, entry); err != nil {
			log.Fatalf("create game sample %q: %v", entry.ID, err)
		}
	}

	productSamples := sampleProducts()
	validateSampleLimit("products", len(productSamples))
	for _, product := range productSamples {
		if err := deleteProductIfExists(ctx, productStore, product.ID); err != nil {
			log.Fatalf("delete product sample %q: %v", product.ID, err)
		}
		if err := deleteProductIfExists(ctx, productStore, product.Slug); err != nil {
			log.Fatalf("delete product sample %q: %v", product.Slug, err)
		}
		if _, err := productStore.Create(ctx, product); err != nil {
			log.Fatalf("create product sample %q: %v", product.Slug, err)
		}
	}
	if _, err := productStore.UpdateSection(ctx, func(settings *products.ProductSectionSettings) error {
		settings.Enabled = true
		settings.Title = "Esoteric Section"
		settings.Description = "Study tools, decks, and small digital products worth keeping close."
		return nil
	}); err != nil {
		log.Fatalf("save products section sample: %v", err)
	}

	introSample := sampleIntro()
	if _, err := introStore.Save(ctx, introSample); err != nil {
		log.Fatalf("save intro sample: %v", err)
	}

	contactProfileSample := sampleContactProfile()
	if _, err := contactStore.SaveProfile(ctx, contactProfileSample); err != nil {
		log.Fatalf("save contact profile sample: %v", err)
	}

	fmt.Printf("seeded %d sample projects, %d sample certificates, %d sample articles, %d sample work experiences, %d sample skill categories, %d sample skills, %d sample links, %d sample tools, %d sample music entries, %d sample series entries, %d sample games, %d sample products, products section, intro, and contact profile\n", len(projectSamples), len(certificateSamples), len(articleSamples), len(workExperienceSamples), len(skillCategorySamples), len(skillSamples), len(linkSamples), len(toolSamples), len(musicSamples), len(seriesSamples), len(gameSamples), len(productSamples))
}

func validateSampleLimit(name string, count int) {
	if count > maxSampleRecords {
		log.Fatalf("sample %s has %d records; max is %d", name, count, maxSampleRecords)
	}
}

func deleteProjectIfExists(ctx context.Context, store projects.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, projects.ErrNotFound) {
		return nil
	}
	return err
}

func deleteCertificateIfExists(ctx context.Context, store certificates.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, certificates.ErrNotFound) {
		return nil
	}
	return err
}

func deleteArticleIfExists(ctx context.Context, store articles.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, articles.ErrNotFound) {
		return nil
	}
	return err
}

func deleteWorkExperienceIfExists(ctx context.Context, store workexperience.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, workexperience.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSkillCategoryIfExists(ctx context.Context, store skills.Store, idOrSlug string) error {
	err := store.DeleteCategory(ctx, idOrSlug)
	if errors.Is(err, skills.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSkillIfExists(ctx context.Context, store skills.Store, id string) error {
	err := store.DeleteSkill(ctx, id)
	if errors.Is(err, skills.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSkillsInCategoryIfExists(ctx context.Context, store skills.Store, categoryIDOrSlug string) error {
	skillsByID, err := store.ListSkills(ctx, skills.SkillListFilter{CategoryID: categoryIDOrSlug})
	if err != nil {
		return err
	}
	skillsBySlug, err := store.ListSkills(ctx, skills.SkillListFilter{Category: categoryIDOrSlug})
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(skillsByID)+len(skillsBySlug))
	for _, skill := range append(skillsByID, skillsBySlug...) {
		if _, ok := seen[skill.ID]; ok {
			continue
		}
		seen[skill.ID] = struct{}{}
		if err := deleteSkillIfExists(ctx, store, skill.ID); err != nil {
			return err
		}
	}

	return nil
}

func deleteLinkIfExists(ctx context.Context, store links.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, links.ErrNotFound) {
		return nil
	}
	return err
}

func deleteToolIfExists(ctx context.Context, store tools.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, tools.ErrNotFound) {
		return nil
	}
	return err
}

func deleteMusicIfExists(ctx context.Context, store music.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, music.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSeriesIfExists(ctx context.Context, store series.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, series.ErrNotFound) {
		return nil
	}
	return err
}

func deleteGameIfExists(ctx context.Context, store games.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, games.ErrNotFound) {
		return nil
	}
	return err
}

func deleteProductIfExists(ctx context.Context, store products.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, products.ErrNotFound) {
		return nil
	}
	return err
}

func sampleProjectIDsForCleanup() []string {
	return []string{
		"sample-portfolio-api-server",
		"portfolio-api-server",
		"sample-portfolio-admin-dashboard",
		"portfolio-admin-dashboard",
		"migrated-project-1",
		"migrated-project-1-latin-tienda",
		"migrated-project-2",
		"migrated-project-2-cyclistian-bicycle-shop",
		"migrated-project-3",
		"migrated-project-3-haru-queue-ordering",
		"migrated-project-4",
		"migrated-project-4-harux-app-ui-ux",
		"migrated-project-5",
		"migrated-project-5-light-of-the-world-worldwide-ministries-site",
		"migrated-project-6",
		"migrated-project-6-crisp-online-ordering",
		"migrated-project-7",
		"migrated-project-7-bootcamp-project",
		"migrated-project-8",
		"migrated-project-8-node-api-boilerplate",
		"migrated-project-9",
		"migrated-project-9-waiterpro-ordering",
		"migrated-project-10",
		"migrated-project-10-churchapp-mobile-ui-ux",
		"migrated-project-11",
		"migrated-project-11-churchadmin-dashboard-wip",
		"migrated-project-12",
		"migrated-project-12-impactify-internal-system-v2",
		"migrated-project-13",
		"migrated-project-13-jpo-contact-center-nexus",
		"migrated-project-14",
		"migrated-project-14-netflix-clone-next13-latest",
	}
}

func sampleProjects() []projects.Project {
	return []projects.Project{
		migratedProject("1", "Latin Tienda", "South American Ecommerce Site", "", "", "laravel, php, node, react", "laravel, react", "/assets/migrated/projects/latintienda/homepage.png", 1, "2023-07-15 03:12:28.416392+00",
			"/assets/migrated/projects/latintienda/categories.png",
			"/assets/migrated/projects/latintienda/eletronic-products.png",
			"/assets/migrated/projects/latintienda/login.png",
			"/assets/migrated/projects/latintienda/product.png",
			"/assets/migrated/projects/latintienda/products.png"),
		migratedProject("2", "Cyclistian - Bicycle Shop", "An e-commerce website for bicycles, bike parts or acccesories.", "", "", "", "", "/assets/migrated/projects/cyclistian/store.png", 2, "2023-07-15 03:13:18.143608+00",
			"/assets/migrated/projects/cyclistian/banner.png",
			"/assets/migrated/projects/cyclistian/orders.png",
			"/assets/migrated/projects/cyclistian/thumbnail.png",
			"/assets/migrated/projects/cyclistian/visitus.png"),
		migratedProject("3", "Haru Queue Ordering", "A real time ordering Progressive Web App where restaurant customers can place their order from the app using the restaurant's provided smartphones or tablets in the table. Their order then will be added to the queue and the kitchen staffs and the admin could monitor the orders of all tables and process it in Queue.", "", "https://harux-queue-ordering.vercel.app/", "", "", "/assets/migrated/projects/haru-queue-ordering/user_menu.png", 3, "2023-07-15 03:14:15.678173+00",
			"/assets/migrated/projects/haru-queue-ordering/admin_chickens.png",
			"/assets/migrated/projects/haru-queue-ordering/admin_dashboard.png",
			"/assets/migrated/projects/haru-queue-ordering/admin_order_detail.png",
			"/assets/migrated/projects/haru-queue-ordering/admin_sauce_categories.png",
			"/assets/migrated/projects/haru-queue-ordering/admin_sauces.png",
			"/assets/migrated/projects/haru-queue-ordering/admin_tables.png",
			"/assets/migrated/projects/haru-queue-ordering/login.png",
			"/assets/migrated/projects/haru-queue-ordering/user_all_orders.png",
			"/assets/migrated/projects/haru-queue-ordering/user_menu_2.png"),
		migratedProject("4", "Harux App UI/UX", "UI / UX Prototype for harux queue ordering application", "", "https://xd.adobe.com/view/f4094e54-9e82-400b-76d9-304a562778a1-8238/", "", "", "/assets/migrated/projects/harux-app-ui-ux/1.png", 5, "2023-07-19 23:25:15.443768+00",
			"/assets/migrated/projects/harux-app-ui-ux/2.png",
			"/assets/migrated/projects/harux-app-ui-ux/3.png",
			"/assets/migrated/projects/harux-app-ui-ux/4.png",
			"/assets/migrated/projects/harux-app-ui-ux/5.png"),
		migratedProject("5", "Light of the World Worldwide Ministries Site", "Website for church organization Light of the World Worldwide Ministries. Get newcomers sign up info, visitor inquiries and pass to the internal system.", "", "", "", "", "/assets/migrated/projects/lowwm/Homepage.png", 7, "2023-07-19 23:25:45.796568+00",
			"/assets/migrated/projects/lowwm/ContactUs.png",
			"/assets/migrated/projects/lowwm/Events.png",
			"/assets/migrated/projects/lowwm/Gallery.png",
			"/assets/migrated/projects/lowwm/Schedule.png",
			"/assets/migrated/projects/lowwm/Sermons.png"),
		migratedProject("6", "Crisp Online Ordering", "Contribution to crisp online ordering, dashboard and mobile apps", "", "https://www.crispqsr.com/", "", "", "/assets/migrated/projects/crisp/crisp-site.png", 6, "2023-07-19 23:26:31.296606+00",
			"/assets/migrated/projects/crisp/locations.png",
			"/assets/migrated/projects/crisp/menu.png"),
		migratedProject("7", "Bootcamp Project", "Bootcamp Challenge Project using an unfamiliar backend framework. Created a cars dealing website where user can see car listing and contact car seller when signed up. Used the project as a training on how to use SCRUM practice.", "", "", "", "", "/assets/migrated/projects/bootcamp-project/homepage.png", 4, "2023-07-19 23:27:03.271084+00",
			"/assets/migrated/projects/bootcamp-project/add_car.png",
			"/assets/migrated/projects/bootcamp-project/car_details_authed.png",
			"/assets/migrated/projects/bootcamp-project/gtr-red-3.jpg",
			"/assets/migrated/projects/bootcamp-project/profile_car_list.png",
			"/assets/migrated/projects/bootcamp-project/profile.png",
			"/assets/migrated/projects/bootcamp-project/sign_up.png",
			"/assets/migrated/projects/bootcamp-project/update_car.png",
			"/assets/migrated/projects/bootcamp-project/view_car_not_authed.png"),
		migratedProject("8", "Node API Boilerplate", "Node Architecture boilerplate to speedup development, example project is 'Bootcamp Directory'", "https://github.com/eralmendral1/node_api_boilerplate", "", "", "", "/assets/migrated/projects/node-api-boilerplate/Thumbnail.png", 11, "2023-07-19 23:28:28.075372+00",
			"/assets/migrated/projects/node-api-boilerplate/Postman.png"),
		migratedProject("9", "Waiterpro Ordering", "Contributed to Waiterpro Online Ordering", "", "https://www.waiterpro.com/", "", "", "/assets/migrated/projects/waiterpro/site.png", 12, "2023-07-19 23:29:22.931019+00",
			"/assets/migrated/projects/waiterpro/dashboard.png"),
		migratedProject("10", "ChurchApp Mobile UI/UX", "UI-UX Mobile Prototype church app", "", "https://xd.adobe.com/view/c67f2f8b-ab1a-43bf-6d73-2232defea040-b04e/grid/", "", "", "/assets/migrated/projects/churchapp-mobile-ui-ux/Wireframes.png", 10, "2023-07-19 23:30:10.798262+00",
			"/assets/migrated/projects/churchapp-mobile-ui-ux/Reports.png"),
		migratedProject("11", "ChurchAdmin Dashboard (WIP)", "Backend Dashboard for managing reports, users, groups, roles, trainings, etc. of the church. Some data are fetched from the CMS and sync to AWS. See architecture diagram in the images slides", "", "https://congregation-suite.vercel.app/dashboard/users", "", "", "/assets/migrated/projects/churchadmin/vips.png", 9, "2023-07-19 23:31:01.20104+00",
			"/assets/migrated/projects/churchadmin/Architecture.png",
			"/assets/migrated/projects/churchadmin/add_user.png",
			"/assets/migrated/projects/churchadmin/users.png"),
		migratedProject("12", "Impactify Internal System v2", "Contribution to Internal System Frontend Development Version2", "", "https://impactify.io/", "", "", "/assets/migrated/projects/impactify/impactify.png", 8, "2023-07-19 23:32:40.963587+00"),
		migratedProject("13", "JPO Contact Center - Nexus", "A call center application for handling real-time Calls, SMS and Tickets. Lead developer of frontend, backend and server, database setup and management.", "", "https://nexus.justpressone.com/", "", "", "/assets/migrated/projects/jpo-contact-center-nexus/nexus.png", 13, "2023-07-19 23:34:05.480017+00"),
		migratedProject("14", "Netflix-Clone Next13@latest", "Netflix clone using Next13, Tailwind, Prisma, MongoDB", "", "https://rt-netflix-clone.vercel.app/auth", "next13", "next13", "/assets/migrated/projects/netflix-clone/nt-netflixclone-home.png", 14, "2023-08-05 14:18:48.159811+00",
			"/assets/migrated/projects/netflix-clone/nt-netflixclone-auth.png"),
	}
}

func sampleCertificateIDsForCleanup() []string {
	return []string{
		"sample-go-api-certificate",
		"go-api-certificate",
		"sample-cloud-deployment-certificate",
		"cloud-deployment-certificate",
		"migrated-certificate-1",
		"introduction-to-containers",
	}
}

func sampleCertificates() []certificates.Certificate {
	createdAt := mustParseCSVTimestamp("2024-08-24 02:44:45.510773+00")
	return []certificates.Certificate{
		{
			ID:            "migrated-certificate-1",
			Slug:          "introduction-to-containers",
			Title:         "Introduction to Containers",
			CredentialURL: "https://dzknfsmeybpmqbixxiva.supabase.co/storage/v1/object/public/certificates/AWS_Intro_To_Containers.pdf",
			Image: &certificates.CertificateImage{
				ID:         "migrated-certificate-1-image",
				URL:        "/assets/migrated/certificates/aws-intro-to-containers.png",
				AltText:    "Introduction to Containers certificate preview",
				Caption:    "Introduction to Containers",
				UploadedAt: createdAt,
			},
			SortOrder: 1,
			Status:    certificates.StatusPublished,
			CreatedAt: createdAt,
		},
	}
}

func migratedProject(sourceID string, title string, description string, githubURL string, demoURL string, tags string, tools string, thumbnail string, sortOrder int, createdAt string, galleryImages ...string) projects.Project {
	parsedCreatedAt := mustParseCSVTimestamp(createdAt)
	cleanTitle := strings.TrimSpace(title)
	project := projects.Project{
		ID:          "migrated-project-" + sourceID,
		Slug:        seedSlugify("migrated project " + sourceID + " " + cleanTitle),
		Title:       cleanTitle,
		Summary:     strings.TrimSpace(description),
		Description: strings.TrimSpace(description),
		TechStack:   splitSeedCSVList(tools),
		Tags:        splitSeedCSVList(tags),
		GitHubURL:   strings.TrimSpace(githubURL),
		DemoURL:     strings.TrimSpace(demoURL),
		SortOrder:   sortOrder,
		Status:      projects.StatusPublished,
		CreatedAt:   parsedCreatedAt,
		PublishedAt: &parsedCreatedAt,
	}

	if strings.TrimSpace(thumbnail) != "" {
		project.MainImage = &projects.ProjectImage{
			ID:         "migrated-project-" + sourceID + "-main-image",
			URL:        strings.TrimSpace(thumbnail),
			AltText:    cleanTitle + " project preview",
			Caption:    cleanTitle,
			SortOrder:  0,
			UploadedAt: parsedCreatedAt,
		}
	}

	for index, imageURL := range galleryImages {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		project.Images = append(project.Images, projects.ProjectImage{
			ID:         fmt.Sprintf("migrated-project-%s-gallery-%d", sourceID, index+1),
			URL:        imageURL,
			AltText:    cleanTitle + " screenshot",
			Caption:    strings.TrimSuffix(path.Base(imageURL), path.Ext(imageURL)),
			SortOrder:  index + 1,
			UploadedAt: parsedCreatedAt,
		})
	}

	return project
}

func sampleArticles() []articles.Article {
	firstPublishedAt := time.Date(2026, time.April, 12, 9, 0, 0, 0, time.UTC)
	secondPublishedAt := time.Date(2026, time.May, 4, 9, 0, 0, 0, time.UTC)

	return []articles.Article{
		{
			ID:            "sample-devto-go-api-routing",
			Title:         "Routing Patterns for Go APIs",
			URL:           "https://dev.to/example/routing-patterns-for-go-portfolio-apis",
			Source:        "Dev.to",
			Summary:       "A practical walkthrough of organizing authenticated CRUD routes in a Go backend.",
			CoverImageURL: "https://picsum.photos/seed/devto-go-api-routing/1200/630",
			Featured:      true,
			SortOrder:     10,
			Status:        articles.StatusPublished,
			PublishedAt:   &firstPublishedAt,
			CreatedAt:     firstPublishedAt,
		},
		{
			ID:            "sample-medium-portfolio-content-models",
			Title:         "Content Models for Preview Cards",
			URL:           "https://medium.com/example/content-models-for-portfolio-preview-cards",
			Source:        "Medium",
			Summary:       "How lightweight metadata can power reusable project, certificate, and article previews.",
			CoverImageURL: "https://picsum.photos/seed/medium-portfolio-content-models/1200/630",
			Featured:      false,
			SortOrder:     20,
			Status:        articles.StatusPublished,
			PublishedAt:   &secondPublishedAt,
			CreatedAt:     secondPublishedAt,
		},
	}
}

func sampleWorkExperienceIDsForCleanup() []string {
	return []string{
		"sample-arete-labs-senior-software-engineer",
		"arete-labs-senior-software-engineer",
		"sample-northstar-systems-backend-engineer",
		"northstar-systems-backend-engineer",
		"migrated-work-experience-1",
		"migrated-work-experience-1-linkage-web-design-and-development-services-web-designer",
		"migrated-work-experience-2",
		"migrated-work-experience-2-brewedlogic-inc-web-developer",
		"migrated-work-experience-3",
		"migrated-work-experience-3-haru-app-developer",
		"migrated-work-experience-4",
		"migrated-work-experience-4-freelance-software-developer",
		"migrated-work-experience-5",
		"migrated-work-experience-5-sitel-call-center-representative",
		"migrated-work-experience-6",
		"migrated-work-experience-6-justpressone-inc-full-stack-developer",
		"migrated-work-experience-7",
		"migrated-work-experience-7-pinzak-networks-back-end-developer",
		"migrated-work-experience-8",
		"migrated-work-experience-8-impactify-front-end-developer",
		"migrated-work-experience-9",
		"migrated-work-experience-9-genpact-it-consultant",
	}
}

func sampleWorkExperiences() []workexperience.WorkExperience {
	return []workexperience.WorkExperience{
		migratedWorkExperience("1", "Web Designer", "Linkage Web Design and Development Services", "2018-11-01", "2019-04-15", "Internship", 3, "https://linkage.ph", "2023-07-19 03:54:47.36328+00", "Turned static designs into lead-focused web pages.", []string{
			"Built responsive website templates from Adobe design files.",
			"Created straightforward pages aimed at converting visitor interest into leads.",
		}),
		migratedWorkExperience("2", "Web Developer", "Brewedlogic Inc", "2019-11-15", "2021-01-01", "Regular", 5, "https://brewedlogic.com", "2023-07-19 03:57:01.151226+00", "Built practical web systems across UI, APIs, and support workflows.", []string{
			"Converted UI/UX designs into working React, Vue, and Angular interfaces.",
			"Built backend APIs with Python, Django, NodeJS, Express, NestJS, PHP, and Laravel.",
			"Maintained and diagnosed production information systems.",
			"Worked across planning, coding, testing, and maintenance in an agile workflow.",
			"Used Git and GitLab for version control and release coordination.",
		}),
		migratedWorkExperience("3", "App Developer", "Haru", "2019-07-15", "2019-08-30", "Contract", 4, "", "2023-07-19 03:58:29.484099+00", "Shipped a real-time ordering prototype into a usable web app.", []string{
			"Built the live ordering flow from prototype to working product.",
			"Created the UI/UX prototype in Adobe XD for client review.",
			"Translated the approved interface into JavaScript application code.",
		}),
		migratedWorkExperience("4", "Software Developer", "Freelance", "2017-01-01", "2019-08-31", "Freelance", 2, "https://ealmendral.vercel.app", "2023-07-19 03:59:52.082606+00", "Handled small-business web builds from design to deployment.", []string{
			"Built JavaScript and CSS user interfaces for web applications.",
			"Developed backend services from the ground up.",
			"Handled hosting, deployment, feature updates, and system setup.",
			"Designed system architecture and databases.",
			"Created UI/UX designs and prototypes with Adobe XD and Photoshop.",
		}),
		migratedWorkExperience("5", "Contact Center Representative", "Foundever (Sitel)", "2016-07-15", "2016-12-30", "Regular", 1, "https://www.linkedin.com/company/sitelgroup", "2023-07-19 04:00:56.881381+00", "Resolved customer issues across voice, email, and chat.", []string{
			"Handled voice support with clear, calm communication.",
			"Resolved email cases with practical next steps.",
			"Answered chat conversations quickly and directly.",
		}),
		migratedWorkExperience("6", "Full-Stack Developer", "JustPressOne Inc.", "2021-06-21", "2023-05-15", "Regular", 8, "https://justpressone.com", "2023-07-19 04:02:30.8022+00", "Built a real-time contact center platform for multi-channel support.", []string{
			"Delivered voice, SMS, chat, email, and real-time ticketing workflows.",
			"Owned backend, frontend, database, server management, optimization, and documentation work.",
			"Used Twilio, PusherJS, Laravel/PHP, NodeJS, and Azure services.",
			"Replaced PusherJS with custom WebSocket services in version 2.",
			"Integrated single sign-on with Microsoft Azure Authentication API.",
			"Managed Azure cloud setup and operations.",
		}),
		migratedWorkExperience("7", "Back-End Developer", "Pinzak Networks", "2021-01-20", "2021-05-15", "Contract", 6, "https://www.pinzak.com", "2023-07-19 04:04:13.539616+00", "Built ecommerce backend services for accounts, payments, and catalog workflows.", []string{
			"Developed web API services for the Latin Tienda ecommerce site.",
			"Built user account and digital wallet flows for recording payments.",
			"Contributed PHP/Laravel and NodeJS backend services.",
			"Implemented backend pagination, filters, and sorting.",
		}),
		migratedWorkExperience("8", "Front-End Developer", "Impactify", "2022-05-01", "2022-08-30", "Contract", 7, "https://impactify.io", "2023-07-19 04:12:21.534478+00", "Built the frontend for a cleaner, faster internal platform.", []string{
			"Developed the frontend for version 2 of the internal system.",
			"Built reusable UI libraries for authentication, tables, charts, and internal pages.",
			"Collaborated with project and backend teams to ship the front-facing system.",
		}),
		migratedWorkExperience("9", "Consultant", "Genpact", "2023-12-15", "", "Full-time", 9, "https://www.genpact.com", "2024-08-24 02:25:06.981404+00", "Improved internal banking systems with better UI, delivery flow, and performance.", []string{
			"Deployed as a developer for Macquarie company systems.",
			"Developed internal systems for bank operations.",
			"Improved user interfaces, DevOps workflows, and performance.",
			"Worked with event-driven architecture, microservices, and micro-frontends.",
		}),
	}
}

func sampleSkillCategoryIDsForCleanup() []string {
	return []string{
		"sample-skill-category-backend-engineering",
		"backend-engineering",
		"sample-skill-category-frontend-engineering",
		"frontend-engineering",
		"sample-skill-category-cloud-devops",
		"cloud-devops",
		"migrated-skill-category-1",
		"frontend",
		"migrated-skill-category-2",
		"backend",
		"migrated-skill-category-3",
		"ui-ux-design",
		"migrated-skill-category-4",
		"devops",
		"migrated-skill-category-5",
		"familiarity",
		"migrated-skill-category-6",
		"automated-testing",
		"migrated-skill-category-7",
		"databases",
		"migrated-skill-category-ai-tools",
		"ai-tools",
		"migrated-skill-category-current-focus",
		"current-focus",
		"migrated-skill-category-ai-assisted-engineering",
		"ai-assisted-engineering",
	}
}

func sampleSkillCategories() []skills.SkillCategory {
	return []skills.SkillCategory{
		migratedSkillCategory("1", "Frontend", 1, "2023-07-19 22:04:15.228236+00"),
		migratedSkillCategory("2", "Backend", 2, "2023-07-19 22:04:26.850767+00"),
		migratedSkillCategory("3", "UI/UX Design", 5, "2023-07-19 22:04:49.869442+00"),
		migratedSkillCategory("4", "DevOps", 3, "2023-07-19 22:04:59.171454+00"),
		migratedSkillCategory("5", "Cloud & Other", 7, "2023-07-19 22:05:23.00957+00"),
		migratedSkillCategory("6", "Testing", 6, "2023-07-19 22:45:16.794502+00"),
		migratedSkillCategory("7", "Databases", 4, "2023-07-19 22:49:44.682249+00"),
		{
			ID:          "migrated-skill-category-ai-assisted-engineering",
			Slug:        "ai-assisted-engineering",
			Name:        "AI-Assisted Engineering",
			Description: "AI-assisted tools, practices, and engineering workflow focus areas.",
			SortOrder:   0,
			Status:      skills.StatusPublished,
			CreatedAt:   mustParseCSVTimestamp("2024-08-24 02:44:45.510773+00"),
		},
	}
}

func sampleSkillIDsForCleanup() []string {
	return []string{
		"sample-skill-api-design",
		"sample-skill-go",
		"sample-skill-postgresql",
		"migrated-skill-1",
		"migrated-skill-2",
		"migrated-skill-3",
		"migrated-skill-4",
		"migrated-skill-5",
		"migrated-skill-6",
		"migrated-skill-7",
		"migrated-skill-8",
		"migrated-skill-9",
		"migrated-skill-10",
		"migrated-skill-11",
		"migrated-skill-12",
		"migrated-skill-13",
		"migrated-skill-14",
		"migrated-skill-15",
		"migrated-skill-16",
		"migrated-skill-17",
		"migrated-skill-18",
		"migrated-skill-19",
		"migrated-skill-20",
		"migrated-skill-21",
		"migrated-skill-22",
		"migrated-skill-23",
		"migrated-skill-24",
		"migrated-skill-25",
		"migrated-skill-26",
		"migrated-skill-27",
		"migrated-skill-28",
		"migrated-skill-29",
		"migrated-skill-30",
		"migrated-skill-31",
		"migrated-skill-32",
		"migrated-skill-33",
		"migrated-skill-34",
		"migrated-skill-35",
		"migrated-skill-36",
		"migrated-skill-37",
		"migrated-skill-38",
		"migrated-skill-39",
		"migrated-skill-40",
		"migrated-skill-41",
		"migrated-skill-42",
		"migrated-skill-43",
		"migrated-skill-44",
		"migrated-skill-45",
		"migrated-skill-ai-codex",
		"migrated-skill-ai-claude",
		"migrated-skill-ai-openai",
		"migrated-skill-ai-chatgpt",
		"migrated-skill-ai-cursor",
		"migrated-skill-ai-github-copilot",
		"current-focus-skill-agentic-engineering",
		"current-focus-skill-codex",
		"current-focus-skill-claude",
		"current-focus-skill-openai",
		"current-focus-skill-chatgpt",
		"current-focus-skill-cursor",
		"current-focus-skill-github-copilot",
		"ai-assisted-engineering-skill-agentic-engineering",
		"ai-assisted-engineering-skill-codex",
		"ai-assisted-engineering-skill-claude",
		"ai-assisted-engineering-skill-rag",
		"ai-assisted-engineering-skill-local-llm",
		"ai-assisted-engineering-skill-automated-testing",
		"ai-assisted-engineering-skill-workflow-automation",
	}
}

func sampleSkills() []skills.Skill {
	return []skills.Skill{
		migratedSkill("1", "React", "1", 1, "2023-07-19 22:23:45.376332+00"),
		migratedSkill("2", "State management", "1", 2, "2023-07-19 22:24:02.270672+00"),
		migratedSkill("3", "Vue", "1", 3, "2023-07-19 22:24:12.053144+00"),
		migratedSkill("4", "CSS, Sass, Tailwind", "1", 4, "2023-07-19 22:24:27.707273+00"),
		migratedSkill("5", "Angular", "1", 5, "2023-07-19 22:24:52.476529+00"),
		migratedSkill("6", "Next.js", "1", 6, "2023-07-19 22:25:30.94493+00"),
		migratedSkill("7", "Nuxt", "1", 7, "2023-07-19 22:25:45.913427+00"),
		migratedSkill("8", "Performance", "1", 8, "2023-07-19 22:26:14.124982+00"),
		migratedSkill("9", "PWAs", "1", 9, "2023-07-19 22:26:29.800911+00"),
		migratedSkill("10", "Node.js", "2", 1, "2023-07-19 22:33:05.358685+00"),
		migratedSkill("11", "REST APIs", "2", 2, "2023-07-19 22:33:17.131626+00"),
		migratedSkill("12", "Laravel/PHP", "2", 3, "2023-07-19 22:33:28.883544+00"),
		migratedSkill("13", "Express", "2", 13, "2023-07-19 22:33:42.202509+00"),
		migratedSkill("14", "NestJS", "2", 14, "2023-07-19 22:33:50.538464+00"),
		migratedSkill("15", "GraphQL", "2", 15, "2023-07-19 22:34:04.135+00"),
		migratedSkill("45", "Python", "2", 16, "2024-08-24 02:44:45.510773+00"),
		migratedSkill("16", "Wireframes", "3", 16, "2023-07-19 22:37:30.238348+00"),
		migratedSkill("17", "Prototyping", "3", 17, "2023-07-19 22:37:43.786248+00"),
		migratedSkill("18", "Adobe XD", "3", 18, "2023-07-19 22:38:23.757277+00"),
		migratedSkill("19", "Color systems", "3", 19, "2023-07-19 22:38:37.836857+00"),
		migratedSkill("20", "Web design", "3", 20, "2023-07-19 22:38:58.537378+00"),
		migratedSkill("21", "UX flows", "3", 21, "2023-07-19 22:39:25.603032+00"),
		migratedSkill("22", "Linux servers", "4", 22, "2023-07-19 22:42:13.255625+00"),
		migratedSkill("23", "Docker", "4", 23, "2023-07-19 22:42:25.425973+00"),
		migratedSkill("24", "Scripts", "4", 24, "2023-07-19 22:42:37.164173+00"),
		migratedSkill("25", "CI", "4", 25, "2023-07-19 22:42:52.35414+00"),
		migratedSkill("26", "CD", "4", 26, "2023-07-19 22:42:59.530829+00"),
		migratedSkill("27", "Logs & monitoring", "4", 27, "2023-07-19 22:43:50.605097+00"),
		migratedSkill("28", "NGINX", "4", 28, "2023-07-19 22:44:30.250109+00"),
		migratedSkill("44", "Kubernetes", "4", 29, "2024-08-24 02:44:45.510773+00"),
		migratedSkill("29", "Unit tests", "6", 29, "2023-07-19 22:46:01.224349+00"),
		migratedSkill("30", "Integration tests", "6", 30, "2023-07-19 22:46:20.277706+00"),
		migratedSkill("31", "E2E tests", "6", 31, "2023-07-19 22:46:30.196754+00"),
		migratedSkill("32", "Jest/RTL", "6", 32, "2023-07-19 22:46:41.035527+00"),
		migratedSkill("33", "Azure", "5", 33, "2023-07-19 22:47:50.592935+00"),
		migratedSkill("34", "AWS", "5", 34, "2023-07-19 22:48:22.591459+00"),
		migratedSkill("35", "Go Language", "5", 35, "2023-07-19 22:49:03.673053+00"),
		migratedSkill("36", "Flutter", "5", 36, "2023-07-19 22:49:21.58145+00"),
		migratedSkill("37", "MySQL", "7", 37, "2023-07-19 22:50:52.733462+00"),
		migratedSkill("38", "PostgreSQL", "7", 38, "2023-07-19 22:51:06.192537+00"),
		migratedSkill("39", "MongoDB", "7", 39, "2023-07-19 22:51:14.656071+00"),
		migratedSkill("40", "GCP Firestore", "7", 40, "2023-07-19 22:51:35.21493+00"),
		migratedSkill("41", "Redis", "7", 41, "2023-07-19 22:51:47.716913+00"),
		migratedSkill("42", "Cypress", "6", 42, "2023-07-20 02:21:13.602774+00"),
		migratedSkill("43", "Enzyme", "6", 43, "2023-07-20 02:21:22.091901+00"),
		aiAssistedEngineeringSkill("agentic-engineering", "Agentic Engineering", 1),
		aiAssistedEngineeringSkill("codex", "Codex", 2),
		aiAssistedEngineeringSkill("claude", "Claude", 3),
		aiAssistedEngineeringSkill("rag", "RAG", 4),
		aiAssistedEngineeringSkill("local-llm", "Local LLM", 5),
		aiAssistedEngineeringSkill("automated-testing", "Automated Testing", 6),
		aiAssistedEngineeringSkill("workflow-automation", "Workflow Automation", 7),
	}
}

func migratedSkillCategory(sourceID string, name string, sortOrder int, createdAt string) skills.SkillCategory {
	return skills.SkillCategory{
		ID:        "migrated-skill-category-" + sourceID,
		Slug:      seedSlugify(name),
		Name:      name,
		SortOrder: sortOrder,
		Status:    skills.StatusPublished,
		CreatedAt: mustParseCSVTimestamp(createdAt),
	}
}

func migratedSkill(sourceID string, name string, categoryID string, sortOrder int, createdAt string) skills.Skill {
	return skills.Skill{
		ID:         "migrated-skill-" + sourceID,
		CategoryID: "migrated-skill-category-" + categoryID,
		Name:       name,
		SortOrder:  sortOrder,
		Status:     skills.StatusPublished,
		CreatedAt:  mustParseCSVTimestamp(createdAt),
	}
}

func aiAssistedEngineeringSkill(sourceID string, name string, sortOrder int) skills.Skill {
	return skills.Skill{
		ID:         "ai-assisted-engineering-skill-" + sourceID,
		CategoryID: "migrated-skill-category-ai-assisted-engineering",
		Name:       name,
		SortOrder:  sortOrder,
		Status:     skills.StatusPublished,
		CreatedAt:  mustParseCSVTimestamp("2024-08-24 02:44:45.510773+00"),
	}
}

func migratedWorkExperience(sourceID string, title string, company string, startedAt string, endedAt string, employmentType string, sortOrder int, companyURL string, createdAt string, description string, responsibilities []string) workexperience.WorkExperience {
	parsedStartedAt := mustParseCSVDate(startedAt)
	parsedEndedAt := mustParseOptionalCSVDate(endedAt)
	parsedCreatedAt := mustParseCSVTimestamp(createdAt)
	description = strings.TrimSpace(description)

	return workexperience.WorkExperience{
		ID:               "migrated-work-experience-" + sourceID,
		Slug:             seedSlugify("migrated work experience " + sourceID + " " + company + " " + title),
		Title:            title,
		Company:          company,
		CompanyURL:       companyURL,
		EmploymentType:   employmentType,
		Summary:          description,
		Description:      description,
		Responsibilities: responsibilities,
		StartedAt:        parsedStartedAt,
		EndedAt:          parsedEndedAt,
		Current:          parsedEndedAt == nil,
		SortOrder:        sortOrder,
		Status:           workexperience.StatusPublished,
		PublishedAt:      &parsedCreatedAt,
		CreatedAt:        parsedCreatedAt,
	}
}

func mustParseCSVTimestamp(value string) time.Time {
	parsed, err := time.Parse("2006-01-02 15:04:05.999999999-07", value)
	if err != nil {
		log.Fatalf("parse csv timestamp %q: %v", value, err)
	}
	return parsed.UTC()
}

func mustParseCSVDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		log.Fatalf("parse csv date %q: %v", value, err)
	}
	return parsed.UTC()
}

func mustParseOptionalCSVDate(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed := mustParseCSVDate(value)
	return &parsed
}

func splitSeedCSVList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, part)
	}
	return values
}

var seedSlugSeparator = regexp.MustCompile(`[^a-z0-9]+`)

func seedSlugify(value string) string {
	slug := strings.Trim(seedSlugSeparator.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
	if slug == "" {
		return "item"
	}
	return slug
}

func sampleLinks() []links.Link {
	return []links.Link{
		migratedLink("1", "Stack Overflow", "https://stackoverflow.com/", 1, "2024-08-24 03:11:29.705816+00"),
		migratedLink("2", "Youtube", "https://youtube.com", 2, "2024-08-24 03:12:15.111039+00"),
		migratedLink("3", "HackerRank", "https://www.hackerrank.com/profile/eralmendral", 3, "2024-08-24 03:12:46.201188+00"),
		migratedLink("4", "Dribbble", "https://dribbble.com/", 4, "2024-08-24 03:13:17.242522+00"),
	}
}

func migratedLink(sourceID string, label string, url string, sortOrder int, createdAt string) links.Link {
	return links.Link{
		ID:        "migrated-link-" + sourceID,
		Label:     strings.TrimSpace(label),
		URL:       strings.TrimSpace(url),
		SortOrder: sortOrder,
		Status:    links.StatusPublished,
		CreatedAt: mustParseCSVTimestamp(createdAt),
	}
}

func sampleLinkIDsForCleanup() []string {
	return []string{
		"sample-link-github",
		"sample-link-linkedin",
		"sample-link-resume",
		"sample-link-stack-overflow",
		"sample-link-youtube",
		"sample-link-hackerrank",
		"sample-link-dribbble",
		"migrated-link-1",
		"migrated-link-2",
		"migrated-link-3",
		"migrated-link-4",
	}
}

func sampleTools() []tools.Tool {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []tools.Tool{
		{
			ID:        "sample-tool-codex",
			Name:      "Codex",
			Category:  "AI Tools",
			Summary:   "Agentic coding for implementation, tests, reviews, and repo maintenance.",
			IconClass: "hugeicons-pro:bot",
			Tags: []string{
				"ai",
				"coding",
				"agent",
			},
			SortOrder: 10,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-claude",
			Name:      "Claude",
			Category:  "AI Tools",
			Summary:   "Reasoning assistant for planning, technical review, and clear writing.",
			IconClass: "hugeicons-pro:sparkles",
			Tags: []string{
				"ai",
				"assistant",
				"reasoning",
			},
			SortOrder: 20,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-openai",
			Name:      "OpenAI",
			Category:  "AI Tools",
			Summary:   "Model platform for assistants, automation, and product AI features.",
			IconClass: "hugeicons-pro:ai-brain-01",
			Tags: []string{
				"ai",
				"models",
				"automation",
			},
			SortOrder: 30,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-cursor",
			Name:      "Cursor",
			Category:  "AI Tools",
			Summary:   "AI editor workflow for quick code changes and codebase navigation.",
			IconClass: "hugeicons-pro:cursor-magic-selection-02",
			Tags: []string{
				"ai",
				"editor",
				"coding",
			},
			SortOrder: 40,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-github-copilot",
			Name:      "GitHub Copilot",
			Category:  "AI Tools",
			Summary:   "Inline coding assistant for everyday implementation support.",
			IconClass: "hugeicons-pro:github",
			Tags: []string{
				"ai",
				"github",
				"coding",
			},
			SortOrder: 50,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-docker",
			Name:      "Docker",
			Category:  "Cloud & DevOps",
			Summary:   "Containerized local services and reproducible deployment workflows.",
			IconClass: "fa-brands fa-docker",
			Tags: []string{
				"containers",
				"devops",
				"local-dev",
			},
			SortOrder: 60,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-jetbrains",
			Name:      "JetBrains",
			Category:  "IDEs & Editors",
			Summary:   "Structured IDE workflow for backend, frontend, refactoring, and tests.",
			IconClass: "hugeicons-pro:code-folder",
			Tags: []string{
				"ide",
				"productivity",
				"engineering",
			},
			SortOrder: 70,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
	}
}

func sampleToolIDsForCleanup() []string {
	return []string{
		"sample-tool-codex",
		"sample-tool-claude",
		"sample-tool-opencode",
		"sample-tool-openai",
		"sample-tool-cursor",
		"sample-tool-github-copilot",
		"sample-tool-google-cloud-platform",
		"sample-tool-google-artifact-registry",
		"sample-tool-github-container-registry",
		"sample-tool-docker",
		"sample-tool-vscode",
		"sample-tool-jetbrains",
		"sample-tool-webstorm",
		"sample-tool-goland",
		"sample-tool-git",
		"sample-tool-github",
		"sample-tool-postman",
		"sample-tool-supabase",
		"sample-tool-vercel",
		"sample-tool-aws-s3",
		"sample-tool-github-actions",
		"sample-tool-blender",
	}
}

func sampleMusic() []music.Music {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []music.Music{
		{
			ID:               "sample-music-night-drive",
			Title:            "Night Drive",
			Artist:           "Aster",
			Album:            "Road Notes",
			SpotifyURL:       "https://open.spotify.com/track/sample-night-drive",
			YouTubeURL:       "https://www.youtube.com/watch?v=sample-night-drive",
			MostlyListenedOn: "2026-05-10",
			Notes:            "A late-night focus track for settling into deep work without losing warmth.",
			SortOrder:        10,
			Status:           music.StatusPublished,
			CreatedAt:        createdAt,
		},
		{
			ID:               "sample-music-morning-signal",
			Title:            "Morning Signal",
			Artist:           "Beacon",
			Album:            "Light Map",
			SpotifyURL:       "https://open.spotify.com/track/sample-morning-signal",
			MostlyListenedOn: "2026-05-09",
			Notes:            "The kind of song that makes ordinary routines feel intentional.",
			SortOrder:        20,
			Status:           music.StatusPublished,
			CreatedAt:        createdAt,
		},
		{
			ID:               "sample-music-quiet-loop",
			Title:            "Quiet Loop",
			Artist:           "Harbor",
			Album:            "Still Water",
			YouTubeURL:       "https://www.youtube.com/watch?v=sample-quiet-loop",
			MostlyListenedOn: "2026-05-08",
			Notes:            "A calm repeat listen for thinking through hard problems.",
			SortOrder:        30,
			Status:           music.StatusPublished,
			CreatedAt:        createdAt,
		},
		{
			ID:               "sample-music-silver-room",
			Title:            "Silver Room",
			Artist:           "Kin",
			Album:            "Interior Weather",
			SpotifyURL:       "https://open.spotify.com/track/sample-silver-room",
			MostlyListenedOn: "2026-05-07",
			Notes:            "A steady listen for design passes and late review sessions.",
			SortOrder:        40,
			Status:           music.StatusPublished,
			CreatedAt:        createdAt,
		},
		{
			ID:               "sample-music-last-train-home",
			Title:            "Last Train Home",
			Artist:           "Northline",
			Album:            "Signals",
			YouTubeURL:       "https://www.youtube.com/watch?v=sample-last-train-home",
			MostlyListenedOn: "2026-05-06",
			Notes:            "A clean closer for winding down after a focused build.",
			SortOrder:        50,
			Status:           music.StatusPublished,
			CreatedAt:        createdAt,
		},
	}
}

func sampleSeries() []series.Series {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []series.Series{
		{
			ID:              "sample-series-the-long-room",
			Title:           "The Long Room",
			Category:        series.CategoryTVSeries,
			Creator:         "North Studio",
			Platform:        "StreamBox",
			WatchURL:        "https://example.com/series/the-long-room",
			MostlyWatchedOn: "2026-05-10",
			Notes:           "A quiet favorite about work, friendship, and ordinary courage.",
			SortOrder:       10,
			Status:          series.StatusPublished,
			CreatedAt:       createdAt,
		},
		{
			ID:              "sample-series-moon-harbor",
			Title:           "Moon Harbor",
			Category:        series.CategoryAnime,
			Creator:         "Blue House",
			Platform:        "Crunchyroll",
			WatchURL:        "https://example.com/series/moon-harbor",
			MostlyWatchedOn: "2026-05-09",
			Notes:           "Soft science fiction with a patient emotional center.",
			SortOrder:       20,
			Status:          series.StatusPublished,
			CreatedAt:       createdAt,
		},
		{
			ID:              "sample-series-home-signals",
			Title:           "Home Signals",
			Category:        series.CategoryTVSeries,
			Creator:         "City Room",
			Platform:        "StreamBox",
			WatchURL:        "https://example.com/series/home-signals",
			MostlyWatchedOn: "2026-05-08",
			Notes:           "A comfort watch about chosen family and showing up.",
			SortOrder:       30,
			Status:          series.StatusPublished,
			CreatedAt:       createdAt,
		},
	}
}

func sampleGames() []games.Game {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []games.Game{
		{
			ID:             "sample-game-starlit-roads",
			Title:          "Starlit Roads",
			Studio:         "North Play",
			Platform:       "PC",
			Genre:          "Adventure",
			StoreURL:       "https://example.com/games/starlit-roads",
			MostlyPlayedOn: "2026-05-10",
			Notes:          "A wandering game for decompressing after long days.",
			SortOrder:      10,
			Status:         games.StatusPublished,
			CreatedAt:      createdAt,
		},
		{
			ID:             "sample-game-garden-tactics",
			Title:          "Garden Tactics",
			Studio:         "Green Tile",
			Platform:       "Switch",
			Genre:          "Strategy",
			StoreURL:       "https://example.com/games/garden-tactics",
			MostlyPlayedOn: "2026-05-09",
			Notes:          "Small decisions that feel satisfying to revisit.",
			SortOrder:      20,
			Status:         games.StatusPublished,
			CreatedAt:      createdAt,
		},
		{
			ID:             "sample-game-kindling",
			Title:          "Kindling",
			Studio:         "Small Fire",
			Platform:       "PC",
			Genre:          "Cozy Simulation",
			StoreURL:       "https://example.com/games/kindling",
			MostlyPlayedOn: "2026-05-08",
			Notes:          "A cozy reset game with patient rituals.",
			SortOrder:      30,
			Status:         games.StatusPublished,
			CreatedAt:      createdAt,
		},
	}
}

func sampleProducts() []products.Product {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, time.May, 10, 10, 0, 0, 0, time.UTC)

	return []products.Product{
		{
			ID:            "sample-product-valuable-anki-deck-from-stranger",
			Slug:          "valuable-anki-deck-from-stranger",
			Title:         "Valuable Anki Deck from Stranger",
			Summary:       "A focused Anki deck for memorizing high-signal ideas from obscure notes.",
			Description:   "A compact deck built for daily spaced repetition, with cards organized around durable concepts, recall prompts, and practical review cadence.",
			CoverImageURL: "https://picsum.photos/seed/valuable-anki-deck/1200/800",
			PriceLabel:    "$19",
			CTALabel:      "Get the deck",
			CTAURL:        "https://example.com/valuable-anki-deck-from-stranger",
			Category:      "Anki Decks",
			Tags: []string{
				"anki",
				"study",
				"spaced-repetition",
			},
			Featured:    true,
			SortOrder:   10,
			Status:      products.StatusPublished,
			PublishedAt: &publishedAt,
			CreatedAt:   createdAt,
		},
	}
}

func sampleIntro() intro.Intro {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return intro.Intro{
		ID:          intro.DefaultID,
		Title:       "Software | AI - Engineer",
		Description: "I build thoughtful software with strong engineering, clean craft, and attention to both details and the bigger picture.",
		ProfilePicture: &intro.IntroImage{
			ID:         "sample-intro-profile-picture",
			URL:        "https://picsum.photos/seed/ai-engineer-profile/1200/1200",
			AltText:    "Eric Almendral portrait for an AI engineering portfolio",
			Caption:    "Eric Almendral, AI engineer",
			UploadedAt: createdAt,
		},
		CreatedAt: createdAt,
	}
}

func sampleContactProfile() contact.Profile {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return contact.Profile{
		ID:          contact.DefaultProfileID,
		WorkEmail:   "work@eralmendral.dev",
		PhoneNumber: "+1 (555) 010-2026",
		CreatedAt:   createdAt,
	}
}
